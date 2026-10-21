package batch_mail

import (
	"billionmail-core/internal/model/entity"
	"billionmail-core/internal/service/domains"
	"billionmail-core/internal/service/mail_service"
	"billionmail-core/internal/service/maillog_stat"
	"billionmail-core/internal/service/public"
	"billionmail-core/internal/service/warmup"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/util/grand"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/panjf2000/ants/v2"
)

var (
	// global task executor map
	taskExecutors      = make(map[int]*TaskExecutor)
	taskExecutorsMutex sync.RWMutex

	// global rate limiter
	//globalLimiter = rate.NewLimiter(rate.Limit(5000), 100)
)

// GetTaskExecutor get task executor
func GetTaskExecutor(taskId int) *TaskExecutor {
	taskExecutorsMutex.RLock()
	defer taskExecutorsMutex.RUnlock()
	return taskExecutors[taskId]
}

// GetOrCreateTaskExecutor get or create task executor
func GetOrCreateTaskExecutor(ctx context.Context, taskId int) *TaskExecutor {
	taskExecutorsMutex.RLock()
	executor, exists := taskExecutors[taskId]
	taskExecutorsMutex.RUnlock()

	if !exists {
		taskExecutorsMutex.Lock()
		defer taskExecutorsMutex.Unlock()
		// double check, avoid race condition
		if executor, exists = taskExecutors[taskId]; !exists {
			executor = NewTaskExecutor(ctx)
			taskExecutors[taskId] = executor
		}
	}
	return executor
}

// RegisterTaskExecutor register task executor
func RegisterTaskExecutor(taskId int, executor *TaskExecutor) {
	taskExecutorsMutex.Lock()
	defer taskExecutorsMutex.Unlock()
	taskExecutors[taskId] = executor
}

// RemoveTaskExecutor remove task executor
func RemoveTaskExecutor(taskId int) {
	taskExecutorsMutex.Lock()
	defer taskExecutorsMutex.Unlock()

	if executor, exists := taskExecutors[taskId]; exists {
		// stop all operations of the executor
		executor.Stop()
		delete(taskExecutors, taskId)
	}
}

// CleanupIdleExecutors cleanup idle executors
func CleanupIdleExecutors() {
	taskExecutorsMutex.Lock()
	defer taskExecutorsMutex.Unlock()

	now := time.Now()
	for id, executor := range taskExecutors {
		if !executor.IsRunning() && now.Sub(executor.lastActivity) > 30*time.Minute {
			executor.Stop()
			delete(taskExecutors, id)
		}
	}
}

// ProcessEmailTasks
func ProcessEmailTasks(ctx context.Context) {
	// get pending tasks
	var tasks []*entity.EmailTask
	err := g.DB().Model("email_tasks").
		Where("task_process IN (0,1)"). // not started or running
		Where("pause", 0). // not paused
		Where("start_time <= ?", time.Now().Unix()). // start time has arrived
		Order("id ASC").
		Scan(&tasks)

	if err != nil {
		g.Log().Error(ctx, "Failed to get pending email tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	//g.Log().Debug(ctx, "Found %d pending email tasks", len(tasks))

	// process each task
	for _, task := range tasks {
		// check if task already has executor and is running
		executor := GetTaskExecutor(task.Id)
		if executor != nil && executor.IsRunning() {
			continue // skip running task
		}

		// create new executor
		newCtx := gctx.New()
		executor = NewTaskExecutor(newCtx)
		RegisterTaskExecutor(task.Id, executor)

		// start task processing
		go func(taskId int) {
			if err := executor.ProcessTask(newCtx); err != nil {
				g.Log().Error(newCtx, "Error processing task %d: %v", taskId, err)
			}
		}(task.Id)
	}
}

// TaskExecutor task executor
type TaskExecutor struct {
	// context and cancel function
	ctx    context.Context
	cancel context.CancelFunc

	// running status
	isRunning    atomic.Bool
	isPaused     atomic.Bool
	lastActivity time.Time

	// task configuration cache
	taskConfig   *entity.EmailTask
	configLoaded time.Time

	spintaxTemplate *SpintaxTemplate

	// rate controller
	rateController *SimpleRateController

	// worker pool
	pool *ants.Pool
	wg   sync.WaitGroup

	// metrics
	sentCount   atomic.Int64
	failedCount atomic.Int64
	startTime   time.Time

	// pause/resume control
	pauseChan  chan struct{}
	resumeChan chan struct{}

	// dynamic rate limiting metrics
	lastStatCheckTime time.Time
}

// SendResult send result
type SendResult struct {
	RecipientID int
	Success     bool
	MessageID   string
	Error       error
}

// NewTaskExecutor create task executor
func NewTaskExecutor(ctx context.Context) *TaskExecutor {
	taskCtx, cancel := context.WithCancel(ctx)

	// Set server IP in context
	serverIP, _ := public.GetServerIP()

	taskCtx = context.WithValue(taskCtx, "serverIP", serverIP)

	executor := &TaskExecutor{
		ctx:               taskCtx,
		cancel:            cancel,
		lastActivity:      time.Now(),
		startTime:         time.Now(),
		pauseChan:         make(chan struct{}, 1),
		resumeChan:        make(chan struct{}, 1),
		rateController:    NewSimpleRateController(1000),
		lastStatCheckTime: time.Now(),
	}

	return executor
}

// IsRunning check if task is running
func (e *TaskExecutor) IsRunning() bool {
	return e.isRunning.Load()
}

// IsPaused check if task is paused
func (e *TaskExecutor) IsPaused() bool {
	return e.isPaused.Load()
}

// ProcessTask
func (e *TaskExecutor) ProcessTask(ctx context.Context) error {
	// prevent duplicate running
	if !e.isRunning.CompareAndSwap(false, true) {
		return errors.New("task executor is already running")
	}

	defer e.isRunning.Store(false)

	// update activity time
	e.lastActivity = time.Now()

	// get task id
	taskId, err := e.getTaskIdFromContext(ctx)
	if err != nil {
		g.Log().Error(ctx, "failed to get task id: %v", err)
		return err
	}

	// get task info
	task, err := GetTaskInfo(ctx, taskId)
	if err != nil {
		return fmt.Errorf("failed to get task info: %w", err)
	}

	if task == nil || task.Id == 0 {
		return fmt.Errorf("task %d not found", taskId)
	}

	// check if task should run
	if task.TaskProcess == 2 { // completed
		return nil
	}

	currentTaskInfo, err := GetTaskInfo(ctx, taskId)
	if err == nil && currentTaskInfo != nil && currentTaskInfo.TaskProcess == 2 {

		return nil
	}

	// set pause status
	if task.Pause == 1 {
		e.isPaused.Store(true)
	}

	// Reset any stranded fetched records from prior interrupted runs (is_sent = 2 -> is_sent = 0)
	_, _ = e.resetFetchedRecords(taskId)

	// check campaign warmup association
	warmupAssociated := false
	if warmupStat, _ := warmup.WarmupCampaign().GetWarmupStatusForCampaign(ctx, int64(taskId)); warmupStat != nil {
		warmupAssociated = true
	}

	e.ctx = context.WithValue(e.ctx, "warmupAssociated", warmupAssociated)

	// configure rate controller
	e.configureRateController(task)

	// get template info
	template, err := e.getTemplateInfo(ctx, task.TemplateId)
	if err != nil {
		g.Log().Error(ctx, "failed to get template: %v", err)
		return fmt.Errorf("failed to get template: %w", err)
	}

	// process email content
	emailContent := e.processEmailContent(ctx, template.Content, task)

	// create worker pool
	poolSize := task.Threads
	if poolSize <= 0 {
		poolSize = 8 // Auto starts with a smooth 8 concurrent workers (tuned up to 16)
	}

	// detailed record thread parameters
	g.Log().Info(ctx, "task %d: create worker pool, size: %d", task.Id, poolSize)

	// increase worker pool options, improve efficiency
	e.pool, err = ants.NewPool(poolSize,
		ants.WithPreAlloc(true),
		ants.WithPanicHandler(func(p interface{}) {
			g.Log().Error(ctx, "Worker panic: %v", p)
		}),
		ants.WithMaxBlockingTasks(poolSize*100), // allow more waiting tasks
		ants.WithNonblocking(false)) // blocking submit can improve stability

	if err != nil {
		g.Log().Error(ctx, "failed to create worker pool: %v", err)
		return fmt.Errorf("failed to create worker pool: %w", err)
	}

	defer e.pool.Release()

	// update task status to running
	if task.TaskProcess == 0 {
		if err := UpdateTaskProcessStatus(ctx, task.Id, 1); err != nil {
			g.Log().Error(ctx, "failed to update task status: %v", err)
			return fmt.Errorf("failed to update task status: %w", err)
		}
	}

	// start time
	startTime := time.Now()
	//g.Log().Info(ctx, "task %d: start processing, start time: %s", task.Id, startTime.Format("2006-01-02 15:04:05"))

	// process task
	if err := e.processTaskRecipients(ctx, task, emailContent); err != nil {
		g.Log().Error(ctx, "failed to process task: %v", err)
		if errors.Is(err, context.Canceled) {
			g.Log().Info(ctx, "task %d is canceled", task.Id)
			return nil
		}
		return err
	}

	// end time and duration
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	sentCount := e.sentCount.Load()

	// calculate average send rate
	var avgRate float64
	if duration.Seconds() > 0 {
		avgRate = float64(sentCount) / duration.Seconds() * 60
	}

	summaryMsg := fmt.Sprintf("task %d: processing completed, end time: %s, total duration: %.2f minutes, total sent: %d, average rate: %.1f emails/minute",
		task.Id, endTime.Format("2006-01-02 15:04:05"),
		duration.Minutes(), sentCount, avgRate)
	g.Log().Info(ctx, summaryMsg)

	currentTask, err := GetTaskInfo(ctx, taskId)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get current task status: %v", err)
		g.Log().Error(ctx, errMsg)
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	if currentTask.TaskProcess == 2 {
		return nil
	}

	completed, err := e.isTaskComplete(ctx, task.Id)
	if err != nil {
		errMsg := fmt.Sprintf("failed to check completion: %v", err)
		g.Log().Error(ctx, errMsg)
		return fmt.Errorf("failed to check completion: %w", err)
	}

	currentTask, err = GetTaskInfo(ctx, taskId)
	if err == nil && currentTask != nil {
		if currentTask.TaskProcess == 2 {

			return nil
		}
	}

	if completed {
		if err := UpdateTaskProcessStatus(ctx, task.Id, 2); err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}
		completeMsg := fmt.Sprintf("task %d is successfully marked as completed", task.Id)
		g.Log().Info(ctx, completeMsg)
		RemoveTaskExecutor(task.Id) // The executor is removed at the end of the task
	}

	return nil
}

// Stop stop task executor
func (e *TaskExecutor) Stop() {
	if e.cancel != nil {
		e.cancel()
	}

	// wait for all work to complete
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	// wait for 3 seconds
	select {
	case <-done:
		// work is completed
	case <-time.After(3 * time.Second):
		// timeout, force stop
	}

	// release worker pool
	if e.pool != nil {
		e.pool.Release()
	}

	e.isRunning.Store(false)
}

// PauseTask
func (e *TaskExecutor) PauseTask(taskId int) error {
	// Always ensure the database and UI reflect the paused state immediately
	if err := UpdateTaskPauseStatus(context.Background(), taskId, true); err != nil {
		return fmt.Errorf("failed to update task pause status: %w", err)
	}

	// If already marked paused in memory (e.g. failure rate cooldown), reset any stranded records and return
	if e.isPaused.Load() {
		resetCount, _ := e.resetFetchedRecords(taskId)
		g.Log().Infof(context.Background(), "Task %d was already paused in memory; database pause status saved, reset %d records", taskId, resetCount)
		return nil
	}

	// set pause status
	e.isPaused.Store(true)

	// Wait for the current batch processing to be completed
	e.waitForCurrentBatch()

	// Reset the records that have been retrieved but not sent (is_sent = 2 -> is_sent = 0)
	resetCount, err := e.resetFetchedRecords(taskId)
	if err != nil {
		e.isPaused.Store(false)
		return fmt.Errorf("failed to reset fetched records: %w", err)
	}

	g.Log().Infof(context.Background(), "Task %d paused successfully, reset %d fetched records", taskId, resetCount)
	return nil
}

func (e *TaskExecutor) waitForCurrentBatch() {

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		g.Log().Debug(context.Background(), "Current batch completed")
	case <-time.After(30 * time.Second):
		g.Log().Warning(context.Background(), "Timeout waiting for current batch to complete")
	}
}

func (e *TaskExecutor) resetFetchedRecords(taskId int) (int64, error) {
	result, err := g.DB().Model("recipient_info").
		Where("task_id", taskId).
		Where("is_sent", 2).
		Data(g.Map{"is_sent": 0}).
		Update()

	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

func (e *TaskExecutor) ResumeTask(taskId int) error {
	// Always ensure database reflects running status (pause = 0, task_process = 1)
	if err := UpdateTaskPauseStatus(context.Background(), taskId, false); err != nil {
		return fmt.Errorf("failed to update task resume status: %w", err)
	}

	// Reload task configuration (to obtain the latest modifications)
	g.Log().Infof(context.Background(), "Task %d: reloading config before resume...", taskId)
	if err := e.reloadTaskConfig(taskId); err != nil {
		g.Log().Errorf(context.Background(), "Failed to reload task config: %v", err)
	}

	// restore running status
	e.isPaused.Store(false)

	// send resume signal
	select {
	case e.resumeChan <- struct{}{}:
		// successfully send resume signal
	default:
		// channel may be full or blocked, recreate and push
		e.resumeChan = make(chan struct{}, 1)
		e.resumeChan <- struct{}{}
	}

	g.Log().Info(context.Background(), "Task %d resumed successfully with updated config", taskId)
	return nil
}

// getTaskIdFromContext
func (e *TaskExecutor) getTaskIdFromContext(ctx context.Context) (int, error) {
	for id, executor := range taskExecutors {
		if executor == e {
			return id, nil
		}
	}
	return 0, errors.New("task id not found in context")
}

// configureRateController initializes the send rate controller with smooth organic defaults
func (e *TaskExecutor) configureRateController(task *entity.EmailTask) {
	var maxPerMinute int
	if task.Threads <= 0 {
		// Smart Auto Mode: Smooth organic pacing spread over business hours (~600m)
		// e.g. 5k recipients -> 25/min; 30k recipients -> 50/min; 80k+ recipients -> 120/min
		targetPerMin := task.RecipientCount / 600
		if targetPerMin < 25 {
			targetPerMin = 25
		}
		if targetPerMin > 120 {
			targetPerMin = 120
		}
		maxPerMinute = targetPerMin
	} else {
		// Manual threads: baseline proportional to threads
		maxPerMinute = task.Threads * 8
		if maxPerMinute < 15 {
			maxPerMinute = 15
		}
		if maxPerMinute > 120 {
			maxPerMinute = 120
		}
	}

	g.Log().Infof(context.Background(), "task %d (recipients: %d): initialize smooth send rate - %d emails per minute, threads: %d (auto=%v)",
		task.Id, task.RecipientCount, maxPerMinute, task.Threads, task.Threads <= 0)
	e.rateController = NewSimpleRateController(maxPerMinute)
}

func (e *TaskExecutor) loadTaskConfig(taskId int) error {
	task, err := GetTaskInfo(context.Background(), taskId)
	if err != nil {
		return fmt.Errorf("failed to load task config: %w", err)
	}

	if task == nil || task.Id == 0 {
		return fmt.Errorf("task %d not found", taskId)
	}

	e.taskConfig = task
	e.configLoaded = time.Now()

	g.Log().Infof(context.Background(), "Task %d: config loaded into cache", taskId)
	return nil
}

func (e *TaskExecutor) reloadTaskConfig(taskId int) error {
	g.Log().Infof(context.Background(), "Task %d: reloading config from database...", taskId)
	return e.loadTaskConfig(taskId)
}

func (e *TaskExecutor) getTaskConfig() *entity.EmailTask {
	return e.taskConfig
}

// processTaskRecipients
func (e *TaskExecutor) processTaskRecipients(ctx context.Context, task *entity.EmailTask, emailContent string) error {
	const batchSize = 50
	var lastId = 0

	// add performance monitoring timer
	statsTicker := time.NewTicker(15 * time.Second)
	defer statsTicker.Stop()

	// record last check time and sent count
	lastCheckTime := time.Now()
	lastSentCount := int64(0)

	// start statistics goroutine
	go func() {
		for {
			select {
			case <-statsTicker.C:
				currentTime := time.Now()
				currentSent := e.sentCount.Load()

				// calculate interval send rate
				elapsedSeconds := currentTime.Sub(lastCheckTime).Seconds()
				sentInInterval := currentSent - lastSentCount
				ratePerMinute := float64(0)
				if elapsedSeconds > 0 {
					ratePerMinute = float64(sentInInterval) / elapsedSeconds * 60
				}

				// get task id
				taskId, _ := e.getTaskIdFromContext(ctx)

				infoMsg := fmt.Sprintf("task %d: performance stats - %.1f seconds sent %d emails, rate: %.1f emails/minute, goroutine pool usage: %d/%d",
					taskId, elapsedSeconds, sentInInterval, ratePerMinute,
					e.pool.Running(), e.pool.Cap())
				g.Log().Info(ctx, infoMsg)

				// update baseline value
				lastCheckTime = currentTime
				lastSentCount = currentSent

			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		// check if context is canceled
		select {
		case <-ctx.Done():
			g.Log().Info(ctx, "context canceled, stop task execution:", ctx.Err())
			return ctx.Err()
		default:
		}

		// check pause status
		taskCheck, _ := GetTaskInfo(ctx, task.Id)
		if (taskCheck != nil && (taskCheck.Pause == 1 || taskCheck.TaskProcess != 1)) || e.isPaused.Load() {
			e.isPaused.Store(true)
			g.Log().Debug(ctx, "Task %d is paused, waiting for resume signal", task.Id)

			// wait for resume signal
			select {
			case <-e.resumeChan:
				// verify it wasn't resumed while still marked paused in DB
				freshTask, _ := GetTaskInfo(ctx, task.Id)
				if freshTask != nil && (freshTask.Pause == 1 || freshTask.TaskProcess != 1) {
					e.isPaused.Store(true)
					continue
				}

				if e.taskConfig != nil {
					g.Log().Infof(ctx, "Using updated task config after resume: addresser=%s, subject=%s, template_id=%d, full_name=%s",
						e.taskConfig.Addresser, e.taskConfig.Subject, e.taskConfig.TemplateId, e.taskConfig.FullName)

					task = e.taskConfig

					template, err := e.getTemplateInfo(ctx, task.TemplateId)
					if err != nil {
						g.Log().Errorf(ctx, "failed to get updated template: %v", err)
					} else {
						emailContent = e.processEmailContent(ctx, template.Content, task)

					}
				}
			case <-ctx.Done():
				g.Log().Info(ctx, "context canceled, stop task execution:", ctx.Err())
				return ctx.Err()
			}
		}

		// Business hours / sending window check (pause at night)
		if waitDur := e.checkBusinessHoursWindow(ctx, task); waitDur > 0 {
			g.Log().Infof(ctx, "task %d: outside business hours sending window. Pausing for night (sleep %v)...", task.Id, waitDur.Round(time.Second))
			select {
			case <-time.After(waitDur):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Dynamic protection check (every 2 minutes)
		if time.Since(e.lastStatCheckTime) >= 2*time.Minute {
			e.lastStatCheckTime = time.Now()
			e.checkAndAdjustRateLimit(ctx, task.Id)
		}

		// Check if fleet daily campaign limit is reached for today
		if isGlobalDailyLimitExceeded(ctx) {
			g.Log().Infof(ctx, "task %d: fleet daily campaign limit reached for today. Sleeping 30 minutes before checking again...", task.Id)
			select {
			case <-time.After(30 * time.Minute):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Check if any healthy senders are available across active IPs
		if task.RotateSenders == 1 {
			mailboxes, err := getAllMailboxes(ctx)
			if err == nil && len(mailboxes) > 0 {
				healthy := filterHealthyMailboxes(ctx, mailboxes)
				if len(healthy) == 0 {
					g.Log().Infof(ctx, "task %d: all active sender IPs reached their daily safe limit or cooling down. Sleeping 30 minutes...", task.Id)
					select {
					case <-time.After(30 * time.Minute):
						continue
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
		}

		// get a batch of recipients to send
		recipients, err := e.getNextRecipientBatch(ctx, task.Id, lastId, batchSize)
		if err != nil {
			return fmt.Errorf("failed to get recipients: %w", err)
		}

		// no more recipients, exit loop
		if len(recipients) == 0 {
			break
		}

		// record batch size
		//g.Log().Debug(ctx, "task %d: got %d recipients to send", task.Id, len(recipients))

		// update last id
		lastId = recipients[len(recipients)-1].Id

		// process this batch of recipients
		if err := e.processRecipientBatch(ctx, task, recipients, emailContent); err != nil {
			return err
		}

		// adjust send rate
		e.rateController.AdjustRate()
	}

	// wait for all tasks to complete
	//g.Log().Info(ctx, "task %d: all recipients processed, waiting for remaining send tasks to complete...", task.Id)
	e.wg.Wait()
	g.Log().Info(ctx, "task %d: all send tasks completed", task.Id)
	return nil
}

// getNextRecipientBatch
func (e *TaskExecutor) getNextRecipientBatch(ctx context.Context, taskId, lastId, batchSize int) ([]*entity.RecipientInfo, error) {
	var recipients []*entity.RecipientInfo

	err := g.DB().Model("recipient_info").
		Where("task_id", taskId).
		Where("is_sent", 0).
		Where("sent_time <= ?", time.Now().Unix()).
		Where("id > ?", lastId).
		Order("id ASC").
		Limit(batchSize).
		Scan(&recipients)

	if err != nil {
		return recipients, err
	}
	if len(recipients) == 0 {
		// If no more un-fetched records from lastId, check if any stranded records exist in is_sent = 2
		strandedCount, _ := g.DB().Model("recipient_info").
			Where("task_id", taskId).
			Where("is_sent", 2).
			Count()
		if strandedCount > 0 {
			_, _ = e.resetFetchedRecords(taskId)
			// Retry query from beginning (lastId = 0)
			return e.getNextRecipientBatch(ctx, taskId, 0, batchSize)
		}
		return recipients, nil
	}

	// Filter out inactive/bounced recipients from this batch
	emails := make([]string, len(recipients))
	for i, r := range recipients {
		emails[i] = r.Recipient
	}

	var inactiveEmails []string
	err = g.DB().Model("bm_contacts").
		WhereIn("email", emails).
		Where("active", 0).
		Fields("email").
		Scan(&inactiveEmails)

	var activeRecipients []*entity.RecipientInfo
	inactiveIds := make([]int, 0)

	if err == nil && len(inactiveEmails) > 0 {
		inactiveMap := make(map[string]bool)
		for _, email := range inactiveEmails {
			inactiveMap[email] = true
		}

		for _, r := range recipients {
			if inactiveMap[r.Recipient] {
				inactiveIds = append(inactiveIds, r.Id)
			} else {
				activeRecipients = append(activeRecipients, r)
			}
		}
	} else {
		activeRecipients = recipients
	}

	// Mark active ones as fetched (is_sent = 2)
	activeIds := make([]int, len(activeRecipients))
	for i, r := range activeRecipients {
		activeIds[i] = r.Id
	}

	if len(activeIds) > 0 {
		_, err = g.DB().Model("recipient_info").
			WhereIn("id", activeIds).
			Data(g.Map{"is_sent": 2}).
			Update()
		if err != nil {
			g.Log().Error(ctx, "Failed to mark recipients as fetched: %v", err)
			return nil, err
		}
	}

	// Mark inactive ones as processed (is_sent = 1) immediately to skip sending
	if len(inactiveIds) > 0 {
		_, err = g.DB().Model("recipient_info").
			WhereIn("id", inactiveIds).
			Data(g.Map{
				"is_sent":   1,
				"sent_time": time.Now().Unix(),
			}).
			Update()
		if err != nil {
			g.Log().Error(ctx, "Failed to mark inactive recipients as processed: %v", err)
		} else {
			g.Log().Infof(ctx, "Skipped sending to %d inactive/bounced recipients", len(inactiveIds))
		}
	}

	// Shuffle activeRecipients in memory to interleave domains for both new and resumed tasks
	if len(activeRecipients) > 1 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(activeRecipients), func(i, j int) {
			activeRecipients[i], activeRecipients[j] = activeRecipients[j], activeRecipients[i]
		})
	}

	return activeRecipients, nil
}

// processRecipientBatch
func (e *TaskExecutor) processRecipientBatch(ctx context.Context, task *entity.EmailTask, recipients []*entity.RecipientInfo, emailContent string) error {
	// create result channel, buffer size same as recipient count
	resultChan := make(chan *SendResult, len(recipients))

	// create wait group to track all send tasks
	var sendWg sync.WaitGroup

	// create a mutex and flag to control channel closure
	var mu sync.Mutex
	channelClosed := false

	// create a safe send function
	safeSend := func(result *SendResult) {
		mu.Lock()
		defer mu.Unlock()
		if !channelClosed {
			select {
			case resultChan <- result:
				// successfully sent
			case <-ctx.Done():
				// context canceled, no more send
			}
		}
	}

	// safe close channel function
	safeClose := func() {
		mu.Lock()
		defer mu.Unlock()
		if !channelClosed {
			channelClosed = true
			close(resultChan)
		}
	}

	// start result processing goroutine
	resultsDone := make(chan struct{})
	go func() {
		e.processSendResults(ctx, resultChan)
		close(resultsDone)
	}()

	// add an initial wait to sendWg so that sendWg.Wait() doesn't return immediately
	// if workers finish before the submission loop finishes
	sendWg.Add(1)

	// start goroutine to close channel
	go func() {
		// wait for all send tasks to complete or context canceled
		sendDone := make(chan struct{})
		go func() {
			sendWg.Wait()
			close(sendDone)
		}()

		select {
		case <-sendDone:
			// all send tasks completed, safe close channel
			safeClose()
		case <-ctx.Done():
			// context canceled, safe close channel
			safeClose()
		}
	}()

	// check global daily cap at batch level
	if isGlobalDailyLimitExceeded(ctx) {
		g.Log().Warning(ctx, "Global daily sending limit reached. Campaign batch deferred.")
		safeClose()
		return nil
	}

	// submit send task for each recipient
	for _, recipient := range recipients {
		// check again if paused or canceled
		if e.isPaused.Load() {
			select {
			case <-e.resumeChan:
				// resumed
			case <-ctx.Done():
				safeClose() // safe close channel
				return ctx.Err()
			}
		}

		select {
		case <-ctx.Done():
			safeClose()
			return ctx.Err()
		default:
		}

		// wait for rate control
		if err := e.rateController.Wait(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				safeClose() // safe close channel
				return err
			}
			// record error but continue
			g.Log().Debugf(ctx, "Rate limit wait error: %v", err)
		}

		// check if recipient is allowed to send with warmup
		if warmupAssociated, ok := e.ctx.Value("warmupAssociated").(bool); ok && warmupAssociated {
			outboundIP := getOutboundIPForRecipient(ctx, task, recipient)
			if allow, waits, _ := warmup.RateLimiter().Allow(ctx, outboundIP, public.GetMailProviderGroup(recipient.Recipient)); !allow {
				g.Log().Debug(ctx, "Rate limit exceeded for recipient %d, wait for %d seconds after retry, skipping", recipient.Id, waits)
				curTime := int(time.Now().Unix())
				sentTime := curTime + (waits * 2)
				if waits <= 0 {
					sentTime = curTime + 60 // default retry in 60s if waits is 0
				}
				_, _ = g.DB().Ctx(ctx).Model("recipient_info").
					Where("id", recipient.Id).
					Data(g.Map{
						"sent_time": sentTime,
						"is_sent":   0,
					}).Update()
				continue
			}
		}

		// create recipient copy to avoid closure problem
		recipientBak := recipient

		// add wait count
		e.wg.Add(1)
		sendWg.Add(1)

		// submit to worker pool
		err := e.pool.Submit(func() {
			defer e.wg.Done()
			defer sendWg.Done()

			// send email (sendEmail handles per-recipient rotated sender selection and personalization)
			result := e.sendEmail(ctx, task, recipientBak, emailContent)

			// record send
			e.rateController.RecordSend()

			// update stats
			if result.Success {
				e.sentCount.Add(1)
			} else {
				e.failedCount.Add(1)
			}

			// safe send result
			safeSend(result)
		})

		if err != nil {
			e.wg.Done()   // reduce wait count
			sendWg.Done() // reduce send wait count

			// create failed result
			failResult := &SendResult{
				RecipientID: recipient.Id,
				Success:     false,
				Error:       fmt.Errorf("failed to submit to worker pool: %w", err),
			}

			// safe send result
			safeSend(failResult)
		}
	}

	// release the initial wait count we added to sendWg
	sendWg.Done()

	// wait for result processing to complete or context canceled
	select {
	case <-resultsDone:
		// result processing completed
		return nil
	case <-ctx.Done():
		// context canceled
		return ctx.Err()
	}
}

// processSendResults
func (e *TaskExecutor) processSendResults(ctx context.Context, resultChan <-chan *SendResult) {
	const batchSize = 50
	const flushInterval = 200 * time.Millisecond

	successResults := make([]*SendResult, 0, batchSize)
	failedIDs := make([]int, 0, batchSize)

	// create ticker to flush results
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	// flush function
	flushUpdates := func() {
		if len(successResults) == 0 && len(failedIDs) == 0 {
			return
		}

		// process success records
		if len(successResults) > 0 {
			// prepare batch update
			now := time.Now().Unix()
			e.lastActivity = time.Now()
			// batch update recipient status
			// prepare batch update SQL
			if len(successResults) > 0 {
				// method 2: use SQL batch update
				ids := make([]interface{}, 0, len(successResults))
				messageIds := make(map[int]string, len(successResults))

				for _, result := range successResults {
					ids = append(ids, result.RecipientID)
					messageIds[result.RecipientID] = result.MessageID
				}

				// step 1: batch update is sent and sent time
				_, err := g.DB().Model("recipient_info").
					WhereIn("id", ids).
					Data(g.Map{
						"is_sent":   1,
						"sent_time": now,
					}).
					Update()

				if err != nil {
					g.Log().Error(ctx, "batch update recipient status failed: %v", err)
				} else {
					// step 2: update each recipient's message ID
					for id, messageID := range messageIds {
						// remove message_id external < >
						messageID = strings.Trim(messageID, "<>")
						_, err := g.DB().Model("recipient_info").
							Where("id", id).
							Data(g.Map{"message_id": messageID}).
							Update()

						if err != nil {
							g.Log().Error(ctx, "update recipient(ID:%d) message ID failed: %v", id, err)
						}
					}

					g.Log().Debug(ctx, "successfully batch updated %d recipient status", len(successResults))
				}
			}

			// clear success results
			successResults = successResults[:0]
		}

		// clear failed records
		if len(failedIDs) > 0 {
			now := time.Now().Unix()
			// 失败的也更新 is_sent 和 sent_time 避免卡住发送状态
			_, err := g.DB().Model("recipient_info").
				WhereIn("id", failedIDs).
				Data(g.Map{
					"is_sent":   1,
					"sent_time": now,
				}).
				Update()

			if err != nil {
				g.Log().Error(ctx, "batch update failed recipients status failed: %v", err)
			} else {
				g.Log().Debug(ctx, "marked %d failed recipients as sent", len(failedIDs))
			}

			failedIDs = failedIDs[:0]
		}
	}

	// main loop
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				// channel closed, process remaining results
				flushUpdates()
				return
			}

			if result.Success {
				successResults = append(successResults, result)
			} else {
				failedIDs = append(failedIDs, result.RecipientID)

				g.Log().Debugf(ctx, "send email to recipient %d failed: %v",
					result.RecipientID, result.Error)
			}

			// reach batch processing size, flush
			if len(successResults)+len(failedIDs) >= batchSize {
				flushUpdates()
			}

		case <-ticker.C:
			// flush periodically
			flushUpdates()

		case <-ctx.Done():
			// context canceled, process remaining results
			flushUpdates()
			return
		}
	}
}

// getTemplateInfo get template info
func (e *TaskExecutor) getTemplateInfo(ctx context.Context, templateId int) (*entity.EmailTemplate, error) {
	var template entity.EmailTemplate

	err := g.DB().Model("email_templates").
		Where("id", templateId).
		Scan(&template)

	if err != nil {
		return nil, err
	}

	if template.Id == 0 {
		return nil, fmt.Errorf("template %d not found", templateId)
	}

	return &template, nil
}

// processEmailContent
func (e *TaskExecutor) processEmailContent(ctx context.Context, content string, task *entity.EmailTask) string {
	// process unsubscribe link
	if task.Unsubscribe == 1 {
		// __UNSUBSCRIBE_URL__  {{ UnsubscribeURL }}
		if !strings.Contains(content, "__UNSUBSCRIBE_URL__") && !strings.Contains(content, "{{ UnsubscribeURL . }}") {
			content = public.AddUnsubscribeButton(content)
		}

		content = strings.ReplaceAll(content, "__UNSUBSCRIBE_URL__", "{{ UnsubscribeURL . }}")
	}

	// Preparse the spintax template
	if e.spintaxTemplate == nil {
		spintaxParser := GetSpintaxParser()
		e.spintaxTemplate = spintaxParser.ParseTemplate(content)
	}

	return content
}

// personalizeEmail personalize email content
func (e *TaskExecutor) personalizeEmail(ctx context.Context, content string, task *entity.EmailTask, recipient *entity.RecipientInfo) (string, string) {

	var contact entity.Contact
	// 优先按任务分组精确匹配，再按创建时间倒序获取最新一条
	q := g.DB().Model("bm_contacts").Where("email", recipient.Recipient)
	if task.GroupId > 0 {
		q = q.Where("group_id", task.GroupId)
	}
	if err := q.OrderDesc("create_time").Limit(1).Scan(&contact); err != nil {
		g.Log().Error(ctx, "get contact info failed: %v", err)
	}

	// If no records are found or the grouping does not match, revert to retrieving only the latest record based on the email address.
	if contact.Id == 0 {
		if err := g.DB().Model("bm_contacts").Where("email", recipient.Recipient).OrderDesc("create_time").Limit(1).Scan(&contact); err != nil {
			g.Log().Error(ctx, "fallback get contact by email failed: %v", err)
		}
	}

	var emailtask entity.EmailTask
	if task != nil {
		emailtask = *task
	}
	if emailtask.Subject == "" || emailtask.TemplateId == 0 {
		if task != nil && task.Id > 0 {
			var dbTask entity.EmailTask
			if err := g.DB().Model("email_tasks").Where("id", task.Id).Scan(&dbTask); err == nil {
				if emailtask.Addresser != "" {
					dbTask.Addresser = emailtask.Addresser
				}
				if emailtask.FullName != "" {
					dbTask.FullName = emailtask.FullName
				}
				emailtask = dbTask
			}
		}
	}

	// Unsubscribe
	var renderedContent, renderedSubject string
	var err error
	engine := GetTemplateEngine()

	if task.Unsubscribe == 1 {
		//domain := domains.GetBaseURLBySender(task.Addresser)
		domain := domains.GetBaseURL()

		var contactGroupId int
		contactGroupId = task.GroupId

		jwtToken, err := GenerateUnsubscribeJWT(
			recipient.Recipient,
			task.TemplateId,
			task.Id,
			contactGroupId,
		)
		if err != nil {
			g.Log().Error(ctx, "generate unsubscribe JWT failed: %v", err)
			jwtToken = ""
		}

		var unsubscribeJumpURL string
		if contactGroupId > 0 {

			unsubscribeJumpURL = fmt.Sprintf("%s/unsubscribe_new.html?jwt=%s",
				domain, jwtToken)

		} else {

			unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe", domain)
			groupURL := fmt.Sprintf("%s/api/unsubscribe/user_group", domain)
			unsubscribeJumpURL = fmt.Sprintf("%s/unsubscribe.html?jwt=%s&email=%s&url_type=%s&url_unsubscribe=%s",
				domain, jwtToken, recipient.Recipient, groupURL, unsubscribeURL)
		}

		// render email content
		renderedContent, err = engine.RenderEmailTemplate(ctx, content, &contact, &emailtask, unsubscribeJumpURL)
		if err != nil {
			g.Log().Error(ctx, "render email content failed: %v", err)
			renderedContent = content
		}

		// render email subject
		renderedSubject, err = engine.RenderEmailTemplate(ctx, emailtask.Subject, &contact, &emailtask, unsubscribeJumpURL)
		if err != nil {
			g.Log().Error(ctx, "render email subject failed: %v", err)
			renderedSubject = emailtask.Subject
		}
	} else {
		// if unsubscribe is not enabled, render email content
		renderedContent, err = engine.RenderEmailTemplate(ctx, content, &contact, &emailtask, "")
		if err != nil {
			g.Log().Error(ctx, "render email content failed: %v", err)
			renderedContent = content
		}

		// render email subject
		renderedSubject, err = engine.RenderEmailTemplate(ctx, emailtask.Subject, &contact, &emailtask, "")
		if err != nil {
			g.Log().Error(ctx, "render email subject failed: %v", err)
			renderedSubject = emailtask.Subject
		}
	}

	// Restore the erroneous variable
	renderedContent = e.restoreErrorVariables(renderedContent)
	renderedSubject = e.restoreErrorVariables(renderedSubject)

	return renderedContent, renderedSubject
}

// restoreErrorVariables 恢复 [__变量__] 为 {{变量}}
func (e *TaskExecutor) restoreErrorVariables(content string) string {
	re := regexp.MustCompile(`\[__([^_]+)__\]`)
	return re.ReplaceAllString(content, "{{$1}}")
}

// sendEmail send email
func (e *TaskExecutor) sendEmail(ctx context.Context, task *entity.EmailTask, recipient *entity.RecipientInfo, content string) *SendResult {
	// check if context is canceled
	select {
	case <-ctx.Done():
		return &SendResult{
			RecipientID: recipient.Id,
			Success:     false,
			Error:       ctx.Err(),
		}
	default:
		// continue execution
	}

	currentTask := task
	if e.taskConfig != nil {
		currentTask = e.taskConfig
	}

	// Create an effective task copy so rotated sender email & display name are available for personalization
	effectiveTask := *currentTask
	senderEmail := currentTask.Addresser
	senderName := currentTask.FullName

	if currentTask.RotateSenders == 1 {
		// Get all mailboxes across all domains for cross-domain rotation
		mailboxes, err := getAllMailboxes(ctx)
		if err != nil {
			g.Log().Warning(ctx, "failed to get all mailboxes: %v, using original sender", err)
		} else if len(mailboxes) > 0 {
			// Filter out mailboxes on IP cooldown or daily limit
			healthyMailboxes := filterHealthyMailboxes(ctx, mailboxes)
			if len(healthyMailboxes) == 0 {
				g.Log().Warningf(ctx, "task %d: no healthy mailboxes available (all IPs reached daily limit or cooling down)", currentTask.Id)
				return &SendResult{
					RecipientID: recipient.Id,
					Success:     false,
					Error:       errors.New("all active IPs reached daily campaign limit or cooling down"),
				}
			}
			// Select mailbox based on recipient ID for consistent rotation
			selected := selectRotatedSender(healthyMailboxes, recipient.Id)
			senderEmail = selected.Username
			if strings.TrimSpace(selected.FullName) != "" {
				senderName = strings.TrimSpace(selected.FullName)
			}
			effectiveTask.Addresser = senderEmail
			effectiveTask.FullName = senderName
		}
	} else {
		// Single sender: verify the sender's domain & IP is active, not cooling down, and not over daily cap
		domain := extractDomain(senderEmail)
		outboundIP := getOutboundIPForDomain(ctx, domain)
		if outboundIP == "" || isIPCoolingDownOrCapped(ctx, outboundIP) {
			g.Log().Warningf(ctx, "task %d: sender %s (IP %s) is inactive, cooling down, or reached daily cap", currentTask.Id, senderEmail, outboundIP)
			return &SendResult{
				RecipientID: recipient.Id,
				Success:     false,
				Error:       fmt.Errorf("sender %s (IP %s) is inactive, cooling down, or reached daily cap", senderEmail, outboundIP),
			}
		}
	}

	// get rendered content and subject using effective rotated sender details
	renderedContent, renderedSubject := e.personalizeEmail(ctx, content, &effectiveTask, recipient)

	sender, err := mail_service.NewEmailSenderWithLocal(senderEmail)
	if err != nil {
		g.Log().Error(ctx, "create email sender failed: %v", err)
		return &SendResult{
			RecipientID: recipient.Id,
			Success:     false,
			Error:       fmt.Errorf("create email sender failed: %w", err),
		}
	}
	defer sender.Close()
	// set message ID
	messageID := sender.GenerateMessageID()

	//Tracking emails
	//baseURL := domains.GetBaseURLBySender(currentTask.Addresser)
	baseURL := domains.GetBaseURL()
	mail_tracker := maillog_stat.NewMailTracker(renderedContent, currentTask.Id, messageID, recipient.Recipient, baseURL)
	if currentTask.TrackClick == 1 {
		mail_tracker.TrackLinks()
	}
	if currentTask.TrackOpen == 1 {
		mail_tracker.AppendTrackingPixel()
	}
	renderedContent = mail_tracker.GetHTML()

	// create email message with rendered subject
	message := mail_service.NewMessage(renderedSubject, renderedContent)
	message.SetMessageID(messageID)

	// set sender display name (use rotated name if rotation is enabled)
	if senderName != "" {
		message.SetRealName(senderName)
	}

	//g.Log().Infof(ctx, "sendEmail - final check before sending: sender=%s, display_name=%s, subject=%s, recipient=%s",
	//	currentTask.Addresser, currentTask.FullName, renderedSubject, recipient.Recipient)

	// send email
	err = sender.Send(message, []string{recipient.Recipient})
	if err != nil {
		g.Log().Error(ctx, "send email to %s failed: %v", recipient.Recipient, err)
		return &SendResult{
			RecipientID: recipient.Id,
			Success:     false,
			Error:       fmt.Errorf("send email failed: %w", err),
		}
	}

	// Record campaign send for this outbound IP and update warmup history
	recordCampaignSendForIP(ctx, senderEmail)

	return &SendResult{
		RecipientID: recipient.Id,
		MessageID:   messageID,
		Success:     true,
		Error:       nil,
	}
}

// sendEmailMock simulates sending an email and records it in the database.
func (e *TaskExecutor) sendEmailMock(ctx context.Context, task *entity.EmailTask, recipient *entity.RecipientInfo, content string) *SendResult {
	// Check if the context is canceled
	select {
	case <-ctx.Done():
		return &SendResult{
			RecipientID: recipient.Id,
			Success:     false,
			Error:       ctx.Err(),
		}
	default:
		// Continue execution
	}

	// Create an effective task copy so rotated sender email & display name are available for personalization
	effectiveTask := *task
	senderEmail := task.Addresser
	senderName := task.FullName

	if task.RotateSenders == 1 {
		// Get all mailboxes across all domains for cross-domain rotation
		mailboxes, err := getAllMailboxes(ctx)
		if err != nil {
			g.Log().Warning(ctx, "failed to get all mailboxes: %v, using original sender", err)
		} else if len(mailboxes) > 0 {
			// Select mailbox based on recipient ID for consistent rotation
			selected := selectRotatedSender(mailboxes, recipient.Id)
			senderEmail = selected.Username
			if strings.TrimSpace(selected.FullName) != "" {
				senderName = strings.TrimSpace(selected.FullName)
			}
			effectiveTask.Addresser = senderEmail
			effectiveTask.FullName = senderName
		}
	}

	// Get the rendered content and subject
	renderedContent, renderedSubject := e.personalizeEmail(ctx, content, &effectiveTask, recipient)

	sender, err := mail_service.NewEmailSenderWithLocal(senderEmail)
	if err != nil {
		g.Log().Error(ctx, "Failed to create email sender: %v", err)
		return &SendResult{
			RecipientID: recipient.Id,
			Success:     false,
			Error:       fmt.Errorf("failed to create email sender: %w", err),
		}
	}
	defer sender.Close()
	// Set message ID
	messageID := sender.GenerateMessageID()

	// Track email
	//baseURL := domains.GetBaseURLBySender(task.Addresser)
	baseURL := domains.GetBaseURL()
	mail_tracker := maillog_stat.NewMailTracker(renderedContent, task.Id, messageID, recipient.Recipient, baseURL)
	if task.TrackClick == 1 {
		mail_tracker.TrackLinks()
	}
	if task.TrackOpen == 1 {
		mail_tracker.AppendTrackingPixel()
	}
	renderedContent = mail_tracker.GetHTML()

	// Create email message with rendered subject
	message := mail_service.NewMessage(renderedSubject, renderedContent)
	message.SetMessageID(messageID)

	// Set sender display name (use rotated name if rotation is enabled)
	if senderName != "" {
		message.SetRealName(senderName)
	}

	// We will create a log entry and save it, instead of sending.
	// This simulates a successful send.
	postfixMessageID := strings.ToUpper("TEST_" + grand.S(11))
	nowMillis := time.Now().UnixMilli()

	// 1. Create MailSender record (use rotated sender if applicable)
	senderRecord := &maillog_stat.MailSender{
		MailRecord: maillog_stat.MailRecord{
			PostfixMessageID: postfixMessageID,
			LogTimeMillis:    nowMillis,
		},
		Sender: senderEmail,
		Size:   int64(len(renderedContent)),
	}
	_, err = g.DB().Model("mailstat_senders").InsertIgnore(senderRecord)
	if err != nil {
		g.Log().Debugf(ctx, "sendEmailMock: failed to insert mailstat_senders: %v", err)
	}

	// 2. Create MailMessageID record
	messageIDRecord := &maillog_stat.MailMessageID{
		MailRecord: maillog_stat.MailRecord{
			PostfixMessageID: postfixMessageID,
			LogTimeMillis:    nowMillis,
		},
		MessageID: strings.Trim(messageID, "<>"),
	}
	_, err = g.DB().Model("mailstat_message_ids").InsertIgnore(messageIDRecord)
	if err != nil {
		g.Log().Debugf(ctx, "sendEmailMock: failed to insert mailstat_message_ids: %v", err)
	}

	// 3. Create MailSendRecord record
	sendRecord := &maillog_stat.MailSendRecord{
		MailRecord: maillog_stat.MailRecord{
			PostfixMessageID: postfixMessageID,
			LogTimeMillis:    nowMillis,
		},
		Recipient:    recipient.Recipient,
		MailProvider: public.GetMailProviderGroup(recipient.Recipient),
		Status:       "sent",
		Delay:        0.1,              // Mock value
		Delays:       "0/0/0.1/0",      // Mock value
		Dsn:          "2.0.0",          // Mock value for successful send
		Relay:        "mock.relay.com", // Mock value
		Description:  "250 2.0.0 OK",   // Mock value
	}

	_, err = g.DB().Model("mailstat_send_mails").Data(sendRecord).Insert()
	if err != nil {
		g.Log().Errorf(ctx, "sendEmailMock: failed to insert mailstat_send_mails: %v", err)
		return &SendResult{
			RecipientID: recipient.Id,
			Success:     false,
			Error:       fmt.Errorf("sendEmailMock: failed to save record: %w", err),
		}
	}

	// 4. Simulate email open and click
	// Simulate a 50% open rate
	if grand.Intn(100) < 50 {
		openTimeMillis := nowMillis + int64(grand.Intn(3600*1000)) // Simulate opening within 1 hour of sending
		_, err = g.DB().Model("mailstat_opened").Insert(g.Map{
			"campaign_id":        task.Id,
			"log_time_millis":    openTimeMillis,
			"recipient":          recipient.Recipient,
			"message_id":         strings.Trim(messageID, "<>"),
			"postfix_message_id": postfixMessageID,
		})
		if err != nil {
			g.Log().Debugf(ctx, "sendEmailMock: failed to insert mailstat_opened: %v", err)
		}

		// If opened, simulate a 20% click rate
		if grand.Intn(100) < 20 {
			clickTimeMillis := openTimeMillis + int64(grand.Intn(600*1000)) // Simulate clicking within 10 minutes of opening
			_, err = g.DB().Model("mailstat_clicked").Insert(g.Map{
				"campaign_id":        task.Id,
				"log_time_millis":    clickTimeMillis,
				"recipient":          recipient.Recipient,
				"message_id":         strings.Trim(messageID, "<>"),
				"postfix_message_id": postfixMessageID,
			})
			if err != nil {
				g.Log().Debugf(ctx, "sendEmailMock: failed to insert mailstat_clicked: %v", err)
			}
		}
	}

	return &SendResult{
		RecipientID: recipient.Id,
		MessageID:   messageID,
		Success:     true,
		Error:       nil,
	}
}

// isTaskComplete check if task is complete
func (e *TaskExecutor) isTaskComplete(ctx context.Context, taskId int) (bool, error) {
	type CountResult struct {
		TotalCount int `json:"total_count"`
		SentCount  int `json:"sent_count"`
	}

	var result CountResult

	err := g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {

		err := tx.Model("recipient_info").
			Fields("COUNT(1) as total_count, SUM(CASE WHEN is_sent = 1 THEN 1 ELSE 0 END) as sent_count").
			Where("task_id", taskId).
			Scan(&result)
		return err
	})

	if err != nil {
		return false, err
	}

	// if there are no recipients, task is not complete
	if result.TotalCount == 0 {
		return false, nil
	}

	// if sent count equals or exceeds total count, task is complete
	return result.SentCount >= result.TotalCount, nil
}

// GetMetrics get execution metrics
func (e *TaskExecutor) GetMetrics() map[string]interface{} {
	duration := time.Since(e.startTime).Seconds()
	sent := e.sentCount.Load()
	failed := e.failedCount.Load()
	total := sent + failed

	var successRate float64

	if total > 0 {
		successRate = float64(sent) / float64(total)
	}

	return map[string]interface{}{
		"sent_count":    sent,
		"failed_count":  failed,
		"total_count":   total,
		"success_rate":  successRate,
		"current_speed": e.rateController.GetCurrentRate(),
		"max_rate":      e.rateController.GetMaxRate(),
		"duration_sec":  duration,
	}
}

func (e *TaskExecutor) UpdateTaskThreads(taskId int, threads int) error {
	// parameter validation
	if threads <= 0 {
		return fmt.Errorf("threads must be greater than zero")
	}

	if threads > 100 {
		return fmt.Errorf("threads must be less than 100")
	}

	// get task info
	task, err := GetTaskInfo(context.Background(), taskId)
	if err != nil {
		return fmt.Errorf("get task info failed: %w", err)
	}

	if task == nil || task.Id == 0 {
		return fmt.Errorf("task %d not found", taskId)
	}

	// record current pool status
	var oldPoolSize int
	var runningWorkers int
	if e.pool != nil {
		oldPoolSize = e.pool.Cap()
		runningWorkers = e.pool.Running()
	}

	// new threads
	newThreads := threads
	// calculate new rate limit - 20 emails per thread per second
	targetSendPerThreadPerSecond := 20
	newRate := newThreads * targetSendPerThreadPerSecond * 60

	// create new rate controller
	e.rateController = NewSimpleRateController(newRate)

	// if task is running, adjust pool size
	if e.pool != nil && e.IsRunning() {
		// if new pool size is not equal to current pool size, create new pool
		if newThreads != oldPoolSize {
			// if request to decrease capacity, but current running number is close to new capacity, output warning
			if newThreads < oldPoolSize && runningWorkers > int(float64(newThreads)*0.8) {
				warningMsg := fmt.Sprintf("task %d: request pool size (%d) is less than current running workers (%d), may cause task queue",
					taskId, newThreads, runningWorkers)
				g.Log().Warning(context.Background(), warningMsg)
			}

			// create new pool
			newPool, err := ants.NewPool(newThreads,
				ants.WithPreAlloc(true),
				ants.WithPanicHandler(func(p interface{}) {
					g.Log().Error(context.Background(), "Worker panic: %v", p)
				}),
				ants.WithMaxBlockingTasks(newThreads*200),
				ants.WithNonblocking(false))

			if err != nil {
				g.Log().Error(context.Background(), "task %d: create new pool failed: %v", taskId, err)
				// even if creating new pool failed, we will still update rate controller
			} else {
				// get old pool reference
				oldPool := e.pool

				// replace with new pool
				e.pool = newPool

				// safely close old pool
				go func(pool *ants.Pool, oldSize int, oldRunning int) {
					// calculate wait time - adjust dynamically based on current active workers
					waitTime := 1 * time.Second
					if oldRunning > 0 {
						// add 1 second for every 10 active workers, minimum 1 second, maximum 10 seconds
						waitSecs := 1 + oldRunning/10
						if waitSecs > 10 {
							waitSecs = 10
						}
						waitTime = time.Duration(waitSecs) * time.Second
					}

					waitInfoMsg := fmt.Sprintf("task %d: wait %d seconds to release old pool (running: %d/%d)",
						taskId, int(waitTime.Seconds()), oldRunning, oldSize)
					g.Log().Info(context.Background(), waitInfoMsg)

					time.Sleep(waitTime)
					pool.Release()
					releaseMsg := fmt.Sprintf("task %d: old pool released", taskId)
					g.Log().Info(context.Background(), releaseMsg)
				}(oldPool, oldPoolSize, runningWorkers)
			}
		} else {
			keepMsg := fmt.Sprintf("task %d: pool size keep unchanged (%d), only adjust rate controller", taskId, oldPoolSize)
			g.Log().Info(context.Background(), keepMsg)
		}
	}

	// update threads in database
	_, err = g.DB().Model("email_tasks").
		Where("id", taskId).
		Data(g.Map{"threads": newThreads}).
		Update()

	if err != nil {

		return fmt.Errorf("task %d: update database threads failed: %w", taskId, err)
	}

	return nil
}

// MailboxInfo represents a mailbox with email and display name
type MailboxInfo struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
}

// getMailboxesByDomain returns all active mailboxes for a given domain
func getMailboxesByDomain(ctx context.Context, domain string) ([]MailboxInfo, error) {
	var mailboxes []MailboxInfo
	err := g.DB().Model("mailbox").
		Where("domain", domain).
		Where("active", 1).
		Order("username ASC").
		Scan(&mailboxes)
	if err != nil {
		return nil, err
	}
	return mailboxes, nil
}

// getAllMailboxes returns all active mailboxes across all domains (excluding admin domain)
func getAllMailboxes(ctx context.Context) ([]MailboxInfo, error) {
	var mailboxes []MailboxInfo
	err := g.DB().Model("mailbox").
		Where("active", 1).
		Where("domain != ?", "b2bprosperity.com").
		Order("username ASC").
		Scan(&mailboxes)
	if err != nil {
		return nil, err
	}
	return mailboxes, nil
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// selectRotatedSender selects a sender from the mailbox list using SHA256-based
// hash distribution. This ensures mailboxes are pseudo-randomly distributed across
// recipients (no batched domain sending), while remaining deterministic so the
// same recipient always gets the same sender (for reply tracking consistency).
func selectRotatedSender(mailboxes []MailboxInfo, recipientIndex int) MailboxInfo {
	if len(mailboxes) == 0 {
		return MailboxInfo{}
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("billionmail-sender-pick-%d", recipientIndex)))
	idx := int(binary.BigEndian.Uint64(h[:8]) % uint64(len(mailboxes)))
	return mailboxes[idx]
}

func getOutboundIPForRecipient(ctx context.Context, task *entity.EmailTask, recipient *entity.RecipientInfo) string {
	senderEmail := task.Addresser
	if task.RotateSenders == 1 {
		mailboxes, err := getAllMailboxes(ctx)
		if err == nil && len(mailboxes) > 0 {
			healthyMailboxes := filterHealthyMailboxes(ctx, mailboxes)
			if len(healthyMailboxes) > 0 {
				selected := selectRotatedSender(healthyMailboxes, recipient.Id)
				senderEmail = selected.Username
			}
		}
	}
	domain := extractDomain(senderEmail)
	if domain != "" {
		return getOutboundIPForDomain(ctx, domain)
	}
	return ""
}

func (e *TaskExecutor) checkAndAdjustRateLimit(ctx context.Context, taskId int) {
	// 0. Check dynamic settings: Is circuit breaker enabled?
	var circuitBreakerEnabled bool = true
	if err := public.OptionsMgrInstance.GetOption(ctx, "circuit_breaker_enabled", &circuitBreakerEnabled); err == nil && !circuitBreakerEnabled {
		return // Circuit breaker is disabled
	}

	// 1. Get configuration thresholds from env
	throttleThresholdStr := public.MustGetDockerEnv("BOUNCE_THROTTLE_THRESHOLD", "0.08") // 8%
	pauseThresholdStr := public.MustGetDockerEnv("BOUNCE_PAUSE_THRESHOLD", "0.15")       // 15%
	throttleFactorStr := public.MustGetDockerEnv("BOUNCE_THROTTLE_FACTOR", "0.50")       // 50%

	// Parse values
	throttleThreshold := 0.08
	pauseThreshold := 0.15
	throttleFactor := 0.50
	if v, err := strconv.ParseFloat(throttleThresholdStr, 64); err == nil {
		throttleThreshold = v
	}
	if v, err := strconv.ParseFloat(pauseThresholdStr, 64); err == nil {
		pauseThreshold = v
	}
	if v, err := strconv.ParseFloat(throttleFactorStr, 64); err == nil {
		throttleFactor = v
	}

	// Dynamic override for threshold from OptionsMgrInstance if set (e.g. 0.10, 0.15, 0.20)
	var customThreshold float64
	if err := public.OptionsMgrInstance.GetOption(ctx, "circuit_breaker_threshold", &customThreshold); err == nil && customThreshold > 0 {
		pauseThreshold = customThreshold
	}

	// 2. Query total sent, true bad email bounces, and soft deferrals in the last 15 minutes for this task
	now := time.Now().Unix()
	timeWindow := int64(900) // 15 minutes

	var stats struct {
		Total       int `json:"total"`
		HardBounces int `json:"hard_bounces"`
		SoftDefers  int `json:"soft_defers"`
	}

	// We join mailstat_send_mails with mailstat_message_ids and recipient_info
	// Differentiating true invalid email bounces from external IP blocks (S3150) and temporary deferrals (4.x.x)
	err := g.DB().Model("mailstat_send_mails sm").
		Ctx(ctx).
		InnerJoin("mailstat_message_ids mi", "sm.postfix_message_id=mi.postfix_message_id").
		InnerJoin("recipient_info r", "mi.message_id=r.message_id").
		Where("r.task_id", taskId).
		Where("sm.log_time >= ?", now-timeWindow).
		Fields("COUNT(*) as total, " +
			"SUM(CASE WHEN (sm.dsn IN ('5.1.1','5.1.2','5.1.3') " +
			"OR sm.description ILIKE '%user unknown%' " +
			"OR sm.description ILIKE '%NoSuchUser%' " +
			"OR sm.description ILIKE '%invalid recipient%' " +
			"OR sm.description ILIKE '%does not exist%' " +
			"OR sm.description ILIKE '%No such user%' " +
			"OR sm.description ILIKE '%mailbox unavailable%') " +
			"AND NOT (sm.description ILIKE '%S3150%' " +
			"OR sm.description ILIKE '%block list%' " +
			"OR sm.description ILIKE '%Local Policy Violation%' " +
			"OR sm.description ILIKE '%Cloudmark%') THEN 1 ELSE 0 END) as hard_bounces, " +
			"SUM(CASE WHEN sm.status = 'deferred' OR sm.dsn LIKE '4.%' THEN 1 ELSE 0 END) as soft_defers").
		Scan(&stats)

	if err != nil {
		g.Log().Warning(ctx, "failed to get campaign dynamic stats: %v", err)
		return
	}

	if stats.Total < 30 {
		return // not enough sample data yet
	}

	hardBounceRate := float64(stats.HardBounces) / float64(stats.Total)
	softDeferRate := float64(stats.SoftDefers) / float64(stats.Total)

	g.Log().Infof(ctx, "Campaign %d: Last 15m stats - sent: %d, hard bounces: %d (%.2f%%), soft defers: %d (%.2f%%)",
		taskId, stats.Total, stats.HardBounces, hardBounceRate*100, stats.SoftDefers, softDeferRate*100)

	// 3. Smart Dynamic Auto-Scaling & PMTA-Style Backoff
	task, _ := GetTaskInfo(ctx, taskId)
	isAuto := (task == nil || task.Threads <= 0)
	currentRate := e.rateController.GetMaxRate()

	if stats.HardBounces >= 8 && hardBounceRate >= pauseThreshold {
		// Hard bounce spike: Circuit breaker activates to protect domain & IP reputation
		g.Log().Warningf(ctx, "Campaign %d HARD bounce rate (%.2f%%) exceeds pause threshold (%.2f%%). Pausing campaign.",
			taskId, hardBounceRate*100, pauseThreshold*100)

		// Set pause state
		e.isPaused.Store(true)

		// Reset any fetched records that weren't sent so they aren't lost
		_, _ = e.resetFetchedRecords(taskId)

		// Persist pause status in database so campaign is visibly and persistently paused in UI & DB
		_ = UpdateTaskPauseStatus(context.Background(), taskId, true)

		go func() {
			time.Sleep(1 * time.Hour)
			// Check if task was manually paused before auto-resuming
			currentTask, _ := GetTaskInfo(context.Background(), taskId)
			if currentTask != nil && (currentTask.Pause == 1 || currentTask.TaskProcess != 1) {
				g.Log().Infof(context.Background(), "Campaign %d was manually paused (pause=%d, process=%d), skipping auto-resume after cooldown", taskId, currentTask.Pause, currentTask.TaskProcess)
				return
			}
			g.Log().Infof(ctx, "Auto-resuming campaign %d after cooldown", taskId)
			e.isPaused.Store(false)
			select {
			case e.resumeChan <- struct{}{}:
			default:
			}
		}()
	} else if (stats.HardBounces >= 3 && hardBounceRate >= throttleThreshold) || (stats.SoftDefers >= 5 && softDeferRate >= 0.20) {
		// Heavy greylisting or bounce pressure: 50% backoff
		throttledRate := int(float64(currentRate) * throttleFactor)
		if throttledRate < 10 {
			throttledRate = 10
		}

		g.Log().Warningf(ctx, "Campaign %d heavy soft deferrals (%.2f%%) or hard bounces (%.2f%%). Dynamic throttling rate to %d/min (was %d/min).",
			taskId, softDeferRate*100, hardBounceRate*100, throttledRate, currentRate)
		e.rateController.SetMaxPerMinute(throttledRate)

		if isAuto && e.pool != nil && e.pool.Cap() > 4 {
			newCap := e.pool.Cap() / 2
			if newCap < 4 {
				newCap = 4
			}
			e.pool.Tune(newCap)
			g.Log().Warningf(ctx, "Campaign %d: Tuned worker pool down to %d threads during pushback.", taskId, newCap)
		}
	} else if stats.SoftDefers >= 5 && softDeferRate >= 0.10 {
		// Mild greylisting: 25% backoff
		throttledRate := int(float64(currentRate) * 0.75)
		if throttledRate < 15 {
			throttledRate = 15
		}

		g.Log().Warningf(ctx, "Campaign %d mild greylisting (%.2f%% defers). Backing off rate to %d/min (was %d/min).",
			taskId, softDeferRate*100, throttledRate, currentRate)
		e.rateController.SetMaxPerMinute(throttledRate)

		if isAuto && e.pool != nil && e.pool.Cap() > 6 {
			e.pool.Tune(6)
		}
	} else if hardBounceRate < 0.02 && softDeferRate < 0.05 {
		// Pristine Deliverability: Dynamic scaling up based on health
		if isAuto {
			acceleratedRate := int(float64(currentRate) * 1.15)
			if acceleratedRate < 25 {
				acceleratedRate = 25
			}
			if acceleratedRate > 350 {
				acceleratedRate = 350
			}

			if acceleratedRate != currentRate {
				g.Log().Infof(ctx, "Campaign %d pristine deliverability (hard: %.2f%%, soft: %.2f%%). Dynamic rate adjusted to %d/min (was %d/min).",
					taskId, hardBounceRate*100, softDeferRate*100, acceleratedRate, currentRate)
				e.rateController.SetMaxPerMinute(acceleratedRate)
			}

			if e.pool != nil && e.pool.Cap() < 24 {
				newCap := e.pool.Cap() + 2
				if newCap > 24 {
					newCap = 24
				}
				e.pool.Tune(newCap)
				g.Log().Infof(ctx, "Campaign %d: Worker pool tuned to %d threads.", taskId, newCap)
			}
		}
	} else {
		// Stable / Balanced: Ensure rate is at safe baseline
		baseline := 25
		if task != nil && task.RecipientCount > 20000 {
			baseline = 50
		}
		if task != nil && task.Threads > 0 {
			baseline = task.Threads * 8
			if baseline > 100 {
				baseline = 100
			}
		}
		if currentRate < baseline {
			g.Log().Infof(ctx, "Campaign %d deliverability healthy (hard: %.2f%%, soft: %.2f%%). Restoring rate towards baseline %d/min.",
				taskId, hardBounceRate*100, softDeferRate*100, baseline)
			e.rateController.SetMaxPerMinute(baseline)
		}
	}

	// 4. Query fail and defer rate per outbound IP and cool down affected IPs
	var ipStats []struct {
		OutboundIp string `json:"outbound_ip"`
		Total      int    `json:"total"`
		Failed     int    `json:"failed"`
		Deferred   int    `json:"deferred"`
	}
	err = g.DB().Model("mailstat_send_mails sm").
		Ctx(ctx).
		InnerJoin("mailstat_senders sn", "sm.postfix_message_id = sn.postfix_message_id").
		InnerJoin("bm_multi_ip_domain d", "substring(sn.sender from '@(.*)$') = d.domain").
		Where("sm.log_time >= ?", now-timeWindow).
		Group("d.outbound_ip").
		Fields("d.outbound_ip, COUNT(*) as total, " +
			"SUM(CASE WHEN sm.status = 'bounced' OR sm.dsn LIKE '5.%' THEN 1 ELSE 0 END) as failed, " +
			"SUM(CASE WHEN sm.status = 'deferred' OR sm.dsn LIKE '4.%' THEN 1 ELSE 0 END) as deferred").
		Scan(&ipStats)

	if err == nil {
		for _, stat := range ipStats {
			if stat.Total >= 15 {
				bounceRate := float64(stat.Failed) / float64(stat.Total)
				deferRate := float64(stat.Deferred) / float64(stat.Total)
				if bounceRate >= pauseThreshold {
					g.Log().Warningf(ctx, "[COOLDOWN] IP %s hard bounce rate (%.2f%%) exceeds pause threshold. Placing on 2-hour cooldown.", stat.OutboundIp, bounceRate*100)
					_ = g.Redis().SetEX(ctx, "ip:cooldown:"+stat.OutboundIp, "1", 7200)
				} else if deferRate >= 0.35 {
					g.Log().Warningf(ctx, "[COOLDOWN] IP %s elevated soft deferral rate (%.2f%%). Cooling down for 15 minutes to let destination queues clear.", stat.OutboundIp, deferRate*100)
					_ = g.Redis().SetEX(ctx, "ip:cooldown:"+stat.OutboundIp, "1", 900)
				}
			}
		}
	}
}

// getOutboundIPForDomain queries the active outbound IP for a given domain from bm_multi_ip_domain.
// Strictly checks active = 1 so quarantined domains return empty.
func getOutboundIPForDomain(ctx context.Context, domain string) string {
	if domain == "" {
		return ""
	}
	val, err := g.DB().Model("bm_multi_ip_domain").
		Ctx(ctx).
		Where("domain = ? AND active = 1", strings.ToLower(domain)).
		Value("outbound_ip")
	if err == nil && !val.IsEmpty() {
		return val.String()
	}
	return ""
}

// getSafeDailyLimitForIP determines the safe daily campaign volume cap for a given IP.
// Starts at 50/day and gradually increases (+30% daily) based on actual emails sent
// and low bounce rates, up to a maximum cap of 4,000 emails/day per IP.
// If an IP experiences elevated bounces (> 5%), its cap automatically reduces by 30%.
// Allows manual override via Redis key "ip:safe_limit:<ip>" or env "DAILY_SEND_LIMIT_PER_IP".
func getSafeDailyLimitForIP(ctx context.Context, ip string) int {
	if ip == "" {
		return 50
	}

	// 1. Manual override via Redis DB 1
	if val, err := g.Redis().Get(ctx, "ip:safe_limit:"+ip); err == nil && !val.IsEmpty() {
		if limit := val.Int(); limit > 0 {
			return limit
		}
	}

	// 2. Max ceiling (default 4000 as requested, or custom env override)
	maxCeiling := 4000
	if envLimit := public.MustGetDockerEnv("DAILY_SEND_LIMIT_PER_IP", ""); envLimit != "" {
		if strings.ToLower(envLimit) == "disable" {
			return 999999
		}
		if limit, err := strconv.Atoi(envLimit); err == nil && limit > 0 {
			maxCeiling = limit
		}
	}

	today := time.Now().Format("20060102")
	dynamicKey := "ip:dynamic_limit:" + ip
	limitVal, err := g.Redis().Get(ctx, dynamicKey)
	currentLimit := 50 // Safe baseline
	if err == nil && !limitVal.IsEmpty() {
		if l := limitVal.Int(); l >= 50 {
			currentLimit = l
		}
	}

	// Check if this IP has already been evaluated for today's volume ramp
	evalDateKey := "ip:limit_evaluated_date:" + ip
	lastEvalDate, _ := g.Redis().Get(ctx, evalDateKey)

	if lastEvalDate.String() != today {
		yesterday := time.Now().AddDate(0, 0, -1).Format("20060102")
		yesterdaySentVal, _ := g.Redis().Get(ctx, fmt.Sprintf("ip:campaign_daily:%s:%s", ip, yesterday))
		yesterdaySent := yesterdaySentVal.Int()

		// Query 24h bounce rate for this specific IP from DB
		var stat struct {
			Total   int `orm:"total"`
			Bounced int `orm:"bounced"`
		}
		_ = g.DB().Model("mailstat_send_mails sm").
			Ctx(ctx).
			InnerJoin("mailstat_senders sn", "sm.postfix_message_id = sn.postfix_message_id").
			InnerJoin("bm_multi_ip_domain d", "substring(sn.sender from '@(.*)$') = d.domain").
			Where("d.outbound_ip = ?", ip).
			Where("sm.log_time >= ?", time.Now().Unix()-86400).
			Fields("COUNT(*) as total, SUM(CASE WHEN sm.status = 'bounced' OR sm.dsn LIKE '5.%' THEN 1 ELSE 0 END) as bounced").
			Scan(&stat)

		bounceRate := 0.0
		if stat.Total >= 15 {
			bounceRate = float64(stat.Bounced) / float64(stat.Total)
		}

		if bounceRate >= 0.05 {
			// Elevated bounces on this IP: throttle back by 30%
			currentLimit = int(float64(currentLimit) * 0.70)
			if currentLimit < 50 {
				currentLimit = 50
			}
			g.Log().Warningf(ctx, "[RAMP] IP %s elevated bounce rate (%.2f%%). Reduced daily limit to %d.", ip, bounceRate*100, currentLimit)
		} else if yesterdaySent >= int(float64(currentLimit)*0.60) {
			// IP successfully sent >= 60% of yesterday's quota with low bounces: ramp up smoothly by 30%
			newLimit := int(float64(currentLimit) * 1.30)
			if newLimit > maxCeiling {
				newLimit = maxCeiling
			}
			currentLimit = newLimit
			g.Log().Infof(ctx, "[RAMP] IP %s clean deliverability (sent %d yesterday, bounce: %.2f%%). Scaled up daily limit to %d.", ip, yesterdaySent, bounceRate*100, currentLimit)
		}

		// Save new limit in Redis and mark evaluated for today
		_, _ = g.Redis().Set(ctx, dynamicKey, strconv.Itoa(currentLimit))
		_, _ = g.Redis().Set(ctx, evalDateKey, today)
	}

	if currentLimit > maxCeiling {
		currentLimit = maxCeiling
	}
	return currentLimit
}

// isIPCoolingDownOrCapped checks if a specific IP is on cooldown or reached its daily campaign limit.
func isIPCoolingDownOrCapped(ctx context.Context, ip string) bool {
	if ip == "" {
		return true
	}
	isCooldown, err := g.Redis().Get(ctx, "ip:cooldown:"+ip)
	if err == nil && !isCooldown.IsEmpty() {
		return true
	}
	today := time.Now().Format("20060102")
	dailyVal, err := g.Redis().Get(ctx, fmt.Sprintf("ip:campaign_daily:%s:%s", ip, today))
	if err == nil && !dailyVal.IsEmpty() {
		if dailyVal.Int() >= getSafeDailyLimitForIP(ctx, ip) {
			return true
		}
	}
	return false
}

// recordCampaignSendForIP increments per-IP daily campaign counter and global campaign count.
func recordCampaignSendForIP(ctx context.Context, senderEmail string) {
	domain := extractDomain(senderEmail)
	if domain == "" {
		return
	}
	outboundIP := getOutboundIPForDomain(ctx, domain)
	if outboundIP == "" {
		return
	}

	today := time.Now().Format("20060102")
	ipKey := fmt.Sprintf("ip:campaign_daily:%s:%s", outboundIP, today)
	_, _ = g.Redis().Incr(ctx, ipKey)
	_, _ = g.Redis().Expire(ctx, ipKey, 172800) // 48h TTL

	// Record active sending day
	_, _ = g.Redis().SAdd(ctx, "ip:campaign_active_days:"+outboundIP, today)

	// Increment global campaign daily count
	globalKey := fmt.Sprintf("campaign:daily_total:%s", today)
	_, _ = g.Redis().Incr(ctx, globalKey)
	_, _ = g.Redis().Expire(ctx, globalKey, 172800)
}

// filterHealthyMailboxes filters mailboxes to only those on active domains (active = 1),
// not on IP cooldown, and not having exceeded their safe daily campaign sending limit.
// NEVER falls back to unfiltered mailboxes.
func filterHealthyMailboxes(ctx context.Context, mailboxes []MailboxInfo) []MailboxInfo {
	if len(mailboxes) == 0 {
		return nil
	}

	// 1. Bulk query all active domains in one SQL query
	type domainMapping struct {
		Domain     string `orm:"domain"`
		OutboundIP string `orm:"outbound_ip"`
	}
	var mappings []domainMapping
	err := g.DB().Model("bm_multi_ip_domain").
		Ctx(ctx).
		Where("active = 1").
		Fields("domain, outbound_ip").
		Scan(&mappings)
	if err != nil || len(mappings) == 0 {
		g.Log().Warningf(ctx, "filterHealthyMailboxes: failed to fetch active domains: %v", err)
		return nil
	}

	domainToIP := make(map[string]string, len(mappings))
	uniqueIPs := make(map[string]bool)
	for _, m := range mappings {
		if m.Domain != "" && m.OutboundIP != "" {
			domainToIP[strings.ToLower(m.Domain)] = m.OutboundIP
			uniqueIPs[m.OutboundIP] = true
		}
	}

	// 2. Pre-evaluate eligibility for each unique active IP
	today := time.Now().Format("20060102")
	eligibleIPs := make(map[string]bool, len(uniqueIPs))
	for ip := range uniqueIPs {
		// Check Redis cooldown
		isCooldown, err := g.Redis().Get(ctx, "ip:cooldown:"+ip)
		if err == nil && !isCooldown.IsEmpty() {
			eligibleIPs[ip] = false
			continue
		}

		// Check daily cap
		limit := getSafeDailyLimitForIP(ctx, ip)
		dailyVal, err := g.Redis().Get(ctx, fmt.Sprintf("ip:campaign_daily:%s:%s", ip, today))
		if err == nil && !dailyVal.IsEmpty() && dailyVal.Int() >= limit {
			eligibleIPs[ip] = false
			continue
		}

		eligibleIPs[ip] = true
	}

	// 3. Filter mailboxes: only retain mailboxes whose domain is active and IP is eligible
	var healthy []MailboxInfo
	for _, m := range mailboxes {
		domain := strings.ToLower(extractDomain(m.Username))
		if domain == "" {
			continue
		}
		ip, exists := domainToIP[domain]
		if !exists || ip == "" {
			// Domain is inactive or quarantined
			continue
		}
		if eligibleIPs[ip] {
			healthy = append(healthy, m)
		}
	}

	return healthy
}

func isGlobalDailyLimitExceeded(ctx context.Context) bool {
	limitEnv := public.MustGetDockerEnv("DAILY_SEND_LIMIT", "")
	if strings.ToLower(limitEnv) == "disable" {
		return false
	}

	var activeIPs []string
	err := g.DB().Model("bm_multi_ip_domain").
		Ctx(ctx).
		Where("active", 1).
		Fields("DISTINCT outbound_ip").
		Scan(&activeIPs)
	if err != nil || len(activeIPs) == 0 {
		return false
	}

	totalDynamicLimit := 0
	for _, ip := range activeIPs {
		totalDynamicLimit += getSafeDailyLimitForIP(ctx, ip)
	}

	if limitEnv != "" {
		if limit, err := strconv.Atoi(limitEnv); err == nil && limit > 0 {
			totalDynamicLimit = limit
		}
	}

	today := time.Now().Format("20060102")
	val, err := g.Redis().Get(ctx, fmt.Sprintf("campaign:daily_total:%s", today))
	count := 0
	if err == nil && !val.IsEmpty() {
		count = val.Int()
	}

	if count >= totalDynamicLimit {
		g.Log().Warningf(ctx, "[LIMIT] Dynamic daily campaign cap reached (%d/%d sent today across %d IPs).", count, totalDynamicLimit, len(activeIPs))
		return true
	}
	return false
}

// checkBusinessHoursWindow checks if current time in the campaign's timezone is within allowed business hours.
// If outside the window, returns the wait duration until morning (capped at 5m for responsiveness).
func (e *TaskExecutor) checkBusinessHoursWindow(ctx context.Context, task *entity.EmailTask) time.Duration {
	if task == nil {
		return 0
	}

	enabled := task.SendingWindowEnabled == 1
	startStr := strings.TrimSpace(task.SendingWindowStart)
	endStr := strings.TrimSpace(task.SendingWindowEnd)
	tzStr := strings.TrimSpace(task.SendingWindowTz)

	// If task doesn't have it explicitly enabled or set, fallback to global bm_options
	if !enabled && task.SendingWindowEnabled == 0 && startStr == "" {
		val, _ := g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_enabled").Value("value")
		if !val.IsEmpty() && val.String() == "1" {
			enabled = true
			if s, _ := g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_start").Value("value"); !s.IsEmpty() {
				startStr = strings.TrimSpace(s.String())
			}
			if s, _ := g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_end").Value("value"); !s.IsEmpty() {
				endStr = strings.TrimSpace(s.String())
			}
			if s, _ := g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_tz").Value("value"); !s.IsEmpty() {
				tzStr = strings.TrimSpace(s.String())
			}
		}
	}

	if !enabled {
		return 0
	}

	if startStr == "" {
		startStr = "08:00"
	}
	if endStr == "" {
		endStr = "18:00"
	}
	if tzStr == "" {
		tzStr = "America/New_York"
	}

	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		loc = time.UTC
	}

	nowInTz := time.Now().In(loc)
	currentMinutes := nowInTz.Hour()*60 + nowInTz.Minute()

	parseMinutes := func(s string) (int, error) {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid time format")
		}
		h, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		m, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		return h*60 + m, nil
	}

	startMinutes, err1 := parseMinutes(startStr)
	endMinutes, err2 := parseMinutes(endStr)
	if err1 != nil || err2 != nil {
		return 0
	}

	if startMinutes < endMinutes {
		// Daytime sending window: e.g. 08:00 (480) to 18:00 (1080)
		if currentMinutes >= startMinutes && currentMinutes < endMinutes {
			return 0 // Inside allowed sending window
		}

		var targetMorning time.Time
		if currentMinutes < startMinutes {
			// Earlier today morning
			targetMorning = time.Date(nowInTz.Year(), nowInTz.Month(), nowInTz.Day(), startMinutes/60, startMinutes%60, 0, 0, loc)
		} else {
			// Tomorrow morning
			targetMorning = time.Date(nowInTz.Year(), nowInTz.Month(), nowInTz.Day()+1, startMinutes/60, startMinutes%60, 0, 0, loc)
		}
		waitDur := targetMorning.Sub(nowInTz)
		if waitDur > 0 {
			// Cap sleep at 5 minutes to remain responsive to pause/stop/settings updates
			if waitDur > 5*time.Minute {
				return 5 * time.Minute
			}
			return waitDur
		}
	}

	return 0
}
