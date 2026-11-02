package batch_mail

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "billionmail-core/api/batch_mail/v1"
	"billionmail-core/internal/model/entity"
	"billionmail-core/internal/service/public"
	"billionmail-core/internal/service/rbac"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// authenticateApiRequest validates Master Admin JWT, active API key, or web UI session.
func authenticateApiRequest(ctx context.Context) error {
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return errors.New("invalid request context")
	}

	// 1. Allow authenticated web session
	if r.Session.MustGet("safe_path_pass", false).Bool() {
		return nil
	}

	// 2. Extract token/key from all standard header & parameter locations
	token := strings.TrimSpace(r.GetHeader("x-api-key"))
	if token == "" {
		token = strings.TrimSpace(r.GetHeader("X-Api-Key"))
	}
	if token == "" {
		token = strings.TrimSpace(r.GetHeader("ApiKey"))
	}
	if token == "" {
		token = strings.TrimSpace(r.GetHeader("Authorization"))
	}
	if token == "" {
		token = strings.TrimSpace(r.Get("api_key").String())
	}
	if token == "" {
		token = strings.TrimSpace(r.Get("token").String())
	}

	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return gerror.NewCode(gcode.CodeNotAuthorized, "API key or Authorization token is required")
	}

	// 3. Match against active api_templates key
	count, err := g.DB().Model("api_templates").Where("api_key", token).Where("active", 1).Count()
	if err == nil && count > 0 {
		return nil
	}

	// 4. Match against Master Admin JWT
	claims, err := rbac.JWT().ParseToken(token)
	if err == nil && claims != nil {
		return nil
	}

	return gerror.NewCode(gcode.CodeNotAuthorized, "Invalid or unauthorized API key / token")
}

// categorizeBounceReason categorizes raw bounce errors and DSN codes into standardized descriptions
func categorizeBounceReason(dsn, rawReason string) string {
	lowerReason := strings.ToLower(rawReason)
	dsnTrim := strings.TrimSpace(dsn)

	if strings.HasPrefix(dsnTrim, "5.1.") || strings.Contains(lowerReason, "user unknown") || strings.Contains(lowerReason, "does not exist") || strings.Contains(lowerReason, "recipient address rejected") || strings.Contains(lowerReason, "no such user") || strings.Contains(lowerReason, "undeliverable") {
		return "Invalid Recipient / Mailbox Does Not Exist (5.1.x)"
	}
	if strings.HasPrefix(dsnTrim, "5.7.") || strings.Contains(lowerReason, "spam") || strings.Contains(lowerReason, "blocked") || strings.Contains(lowerReason, "blacklist") || strings.Contains(lowerReason, "reputation") || strings.Contains(lowerReason, "policy violation") || strings.Contains(lowerReason, "poor reputation") || strings.Contains(lowerReason, "spamhaus") || strings.Contains(lowerReason, "barracuda") {
		return "Spam Filter / Reputation / Policy Rejection (5.7.x)"
	}
	if strings.HasPrefix(dsnTrim, "4.2.") || strings.HasPrefix(dsnTrim, "4.7.") || strings.Contains(lowerReason, "rate limit") || strings.Contains(lowerReason, "too many") || strings.Contains(lowerReason, "throttl") || strings.Contains(lowerReason, "try again later") {
		return "Rate Limited / Throttled (4.x.x)"
	}
	if strings.HasPrefix(dsnTrim, "5.4.") || strings.HasPrefix(dsnTrim, "5.1.2") || strings.Contains(lowerReason, "dns") || strings.Contains(lowerReason, "domain not found") || strings.Contains(lowerReason, "host not found") || strings.Contains(lowerReason, "no mx") {
		return "Domain / DNS / MX Resolution Failure (5.4.x)"
	}
	if strings.HasPrefix(dsnTrim, "5.2.2") || strings.HasPrefix(dsnTrim, "4.2.2") || strings.Contains(lowerReason, "quota") || strings.Contains(lowerReason, "full") || strings.Contains(lowerReason, "storage") {
		return "Mailbox Full / Quota Exceeded (5.2.2)"
	}
	if strings.HasPrefix(dsnTrim, "4.4.") || strings.Contains(lowerReason, "connection timed out") || strings.Contains(lowerReason, "network is unreachable") || strings.Contains(lowerReason, "connect to") {
		return "Network / Connection Timeout (4.4.x)"
	}
	if strings.HasPrefix(dsnTrim, "5.3.") || strings.Contains(lowerReason, "greylist") {
		return "Greylisted / Temporary Rejection (4.3.x/5.3.x)"
	}
	if dsnTrim != "" && dsnTrim != "unknown" {
		return fmt.Sprintf("Other Delivery Failure (%s)", dsnTrim)
	}
	return "Other Delivery Failure"
}

// CampaignStats provides comprehensive metrics: total sent/succeeded/failed, daily breakdown, bounce reasons, domain stats, and samples.
func (c *ControllerV1) CampaignStats(ctx context.Context, req *v1.CampaignStatsReq) (res *v1.CampaignStatsRes, err error) {
	if err := authenticateApiRequest(ctx); err != nil {
		res = &v1.CampaignStatsRes{}
		res.Code = 401
		res.SetError(err)
		return res, nil
	}

	return c.processCampaignStats(ctx, req.CampaignId, req.SendingDomain, req.Domain, req.StartTime, req.EndTime, req.StartDate, req.EndDate, req.FailedLimit)
}

// CampaignStatsAlias alias route for /batch_mail/api/campaign_stats
func (c *ControllerV1) CampaignStatsAlias(ctx context.Context, req *v1.CampaignStatsAliasReq) (res *v1.CampaignStatsRes, err error) {
	if err := authenticateApiRequest(ctx); err != nil {
		res = &v1.CampaignStatsRes{}
		res.Code = 401
		res.SetError(err)
		return res, nil
	}

	return c.processCampaignStats(ctx, req.CampaignId, req.SendingDomain, req.Domain, req.StartTime, req.EndTime, req.StartDate, req.EndDate, req.FailedLimit)
}

func (c *ControllerV1) processCampaignStats(ctx context.Context, campaignId int64, sendingDomain, domain string, startTime, endTime int64, startDate, endDate string, failedLimit int) (res *v1.CampaignStatsRes, err error) {
	res = &v1.CampaignStatsRes{}

	r := g.RequestFromCtx(ctx)

	// Clean sending domain filter strictly from URL query parameter
	var domainFilter string
	if r != nil {
		qDomain := strings.TrimSpace(r.GetQuery("sending_domain").String())
		if qDomain == "" {
			qDomain = strings.TrimSpace(r.GetQuery("domain").String())
		}
		if qDomain != "" {
			domainFilter = strings.ToLower(strings.TrimPrefix(qDomain, "@"))
		}
	}
	if domainFilter == "" && sendingDomain != "" && strings.Contains(sendingDomain, ".") {
		domainFilter = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(sendingDomain), "@"))
	}
	if domainFilter == "" && domain != "" && strings.Contains(domain, ".") {
		domainFilter = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(domain), "@"))
	}
	if domainFilter != "" && !strings.Contains(domainFilter, ".") {
		domainFilter = ""
	}

	if campaignId == 0 && r != nil {
		campaignId = r.GetQuery("campaign_id").Int64()
		if campaignId == 0 {
			campaignId = r.GetQuery("id").Int64()
		}
	}

	var task *entity.EmailTask
	if campaignId > 0 {
		var t entity.EmailTask
		err = g.DB().Model("email_tasks").Where("id", campaignId).Scan(&t)
		if err != nil || t.Id == 0 {
			res.Code = 404
			res.SetError(gerror.New(fmt.Sprintf("Campaign ID %d not found", campaignId)))
			return res, nil
		}
		task = &t
	}

	// Resolve date ranges
	isCustomDateRange := false
	if startDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.UTC); err == nil {
			startTime = t.Unix()
			isCustomDateRange = true
		}
	}
	if endDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.UTC); err == nil {
			endTime = t.Unix() + 86399
			isCustomDateRange = true
		}
	}

	if startTime <= 0 {
		if task != nil && task.StartTime > 0 {
			startTime = int64(task.StartTime)
		} else {
			startTime = time.Now().Add(-30 * 24 * time.Hour).Unix()
		}
	}
	if endTime <= 0 {
		endTime = time.Now().Unix()
	}
	if startTime > endTime {
		startTime, endTime = endTime, startTime
	}

	if failedLimit <= 0 {
		failedLimit = 50
	} else if failedLimit > 200 {
		failedLimit = 200
	}

	res.Data.Filter = v1.CampaignStatsFilter{
		CampaignId:    campaignId,
		SendingDomain: domainFilter,
		StartTime:     startTime,
		EndTime:       endTime,
		StartDate:     time.Unix(startTime, 0).UTC().Format("2006-01-02"),
		EndDate:       time.Unix(endTime, 0).UTC().Format("2006-01-02"),
	}

	// Populate Campaign Info
	if task != nil {
		statusText := "running"
		if task.Pause == 1 {
			statusText = "paused"
		} else if task.TaskProcess == 0 {
			statusText = "draft"
		} else if task.TaskProcess == 2 {
			statusText = "completed"
		} else if task.TaskProcess == 3 {
			statusText = "paused"
		}

		res.Data.Campaign = &v1.CampaignDetailItem{
			Id:                   task.Id,
			TaskName:             task.TaskName,
			Subject:              task.Subject,
			Addresser:            task.Addresser,
			RecipientCount:       task.RecipientCount,
			SendsCount:           task.SendsCount,
			DeliveredCount:       task.DeliveredCount,
			BouncedCount:         task.BouncedCount,
			DeferredCount:        task.DeferredCount,
			TaskProcess:          task.TaskProcess,
			StatusText:           statusText,
			IsPaused:             task.Pause == 1 || task.TaskProcess == 3,
			StartTime:            int64(task.StartTime),
			StartTimeFormatted:   time.Unix(int64(task.StartTime), 0).UTC().Format("2006-01-02 15:04:05 UTC"),
			SendingWindowEnabled: task.SendingWindowEnabled,
			SendingWindowStart:   task.SendingWindowStart,
			SendingWindowEnd:     task.SendingWindowEnd,
			SendingWindowTz:      task.SendingWindowTz,
		}
	}

	startTimeMs := startTime * 1000
	endTimeMs := (endTime + 1) * 1000

	// CTE materialization string for campaign joins
	cteSql := ""
	if campaignId > 0 {
		var timeFilterSql string
		if isCustomDateRange {
			timeFilterSql = fmt.Sprintf(" AND sent_time >= %d AND sent_time <= %d", startTime, endTime)
		}
		var domainFilterSql string
		if domainFilter != "" {
			domainFilterSql = fmt.Sprintf(" AND split_part(message_id, '@', 2) = '%s'", strings.ReplaceAll(domainFilter, "'", "''"))
		}
		cteSql = fmt.Sprintf(`
			WITH cr AS MATERIALIZED (
				SELECT message_id, split_part(message_id, '@', 2) as sending_domain, recipient, sent_time
				FROM recipient_info
				WHERE task_id = %d AND is_sent = 1 AND message_id != ''%s%s
			)
		`, campaignId, timeFilterSql, domainFilterSql)
	}

	// 1. Core Summary Metrics
	if campaignId > 0 && domainFilter == "" && !isCustomDateRange && task != nil {
		// Use instantaneous precomputed aggregates from email_tasks
		sends := task.SendsCount
		delivered := task.DeliveredCount
		bounced := task.BouncedCount
		deferred := task.DeferredCount
		recipients := task.RecipientCount

		openedCount, _ := g.DB().Model("mailstat_opened").
			Where("campaign_id", campaignId).
			WhereGTE("log_time_millis", startTimeMs).
			WhereLTE("log_time_millis", endTimeMs).
			Fields("count(distinct postfix_message_id) as opened").
			Value()
		clickedCount, _ := g.DB().Model("mailstat_clicked").
			Where("campaign_id", campaignId).
			WhereGTE("log_time_millis", startTimeMs).
			WhereLTE("log_time_millis", endTimeMs).
			Fields("count(distinct postfix_message_id) as clicked").
			Value()

		opened := openedCount.Int()
		clicked := clickedCount.Int()

		var deliveryRate, bounceRate, openRate, clickRate float64
		if sends > 0 {
			deliveryRate = public.Round(float64(delivered)/float64(sends)*100, 2)
			bounceRate = public.Round(float64(bounced)/float64(sends)*100, 2)
			openRate = public.Round(float64(opened)/float64(sends)*100, 2)
			clickRate = public.Round(float64(clicked)/float64(sends)*100, 2)
		}

		res.Data.Summary = v1.CampaignSummaryMetric{
			TotalRecipients: recipients,
			TotalSent:       sends,
			Delivered:       delivered,
			Bounced:         bounced,
			Deferred:        deferred,
			Opened:          opened,
			Clicked:         clicked,
			DeliveryRate:    deliveryRate,
			BounceRate:      bounceRate,
			OpenRate:        openRate,
			ClickRate:       clickRate,
		}
	} else if campaignId > 0 {
		summarySql := cteSql + `
			SELECT 
				count(*) as sends,
				coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
				coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced,
				coalesce(sum(case when sm.status='deferred' then 1 else 0 end), 0) as deferred
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
		`
		row, qErr := g.DB().Query(ctx, summarySql)
		if qErr == nil && len(row) > 0 {
			sends := row[0]["sends"].Int()
			delivered := row[0]["delivered"].Int()
			bounced := row[0]["bounced"].Int()
			deferred := row[0]["deferred"].Int()

			recipients := 0
			if task != nil {
				recipients = task.RecipientCount
			}

			openedCount, _ := g.DB().Model("mailstat_opened").
				Where("campaign_id", campaignId).
				WhereGTE("log_time_millis", startTimeMs).
				WhereLTE("log_time_millis", endTimeMs).
				Fields("count(distinct postfix_message_id) as opened").
				Value()
			clickedCount, _ := g.DB().Model("mailstat_clicked").
				Where("campaign_id", campaignId).
				WhereGTE("log_time_millis", startTimeMs).
				WhereLTE("log_time_millis", endTimeMs).
				Fields("count(distinct postfix_message_id) as clicked").
				Value()

			opened := openedCount.Int()
			clicked := clickedCount.Int()

			var deliveryRate, bounceRate, openRate, clickRate float64
			if sends > 0 {
				deliveryRate = public.Round(float64(delivered)/float64(sends)*100, 2)
				bounceRate = public.Round(float64(bounced)/float64(sends)*100, 2)
				openRate = public.Round(float64(opened)/float64(sends)*100, 2)
				clickRate = public.Round(float64(clicked)/float64(sends)*100, 2)
			}

			res.Data.Summary = v1.CampaignSummaryMetric{
				TotalRecipients: recipients,
				TotalSent:       sends,
				Delivered:       delivered,
				Bounced:         bounced,
				Deferred:        deferred,
				Opened:          opened,
				Clicked:         clicked,
				DeliveryRate:    deliveryRate,
				BounceRate:      bounceRate,
				OpenRate:        openRate,
				ClickRate:       clickRate,
			}
		}
	} else {
		// Global fleet metrics across all campaigns
		summarySql := `
			SELECT 
				count(*) as sends,
				coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
				coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced,
				coalesce(sum(case when sm.status='deferred' then 1 else 0 end), 0) as deferred
			FROM mailstat_send_mails sm
			WHERE sm.log_time_millis >= ? AND sm.log_time_millis <= ?
		`
		args := []interface{}{startTimeMs, endTimeMs}
		if domainFilter != "" {
			summarySql = `
				SELECT 
					count(*) as sends,
					coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
					coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced,
					coalesce(sum(case when sm.status='deferred' then 1 else 0 end), 0) as deferred
				FROM mailstat_send_mails sm
				JOIN mailstat_senders s ON sm.postfix_message_id = s.postfix_message_id
				WHERE sm.log_time_millis >= ? AND sm.log_time_millis <= ?
				  AND lower(split_part(s.sender, '@', 2)) = ?
			`
			args = append(args, domainFilter)
		}

		row, qErr := g.DB().Query(ctx, summarySql, args...)
		if qErr == nil && len(row) > 0 {
			sends := row[0]["sends"].Int()
			delivered := row[0]["delivered"].Int()
			bounced := row[0]["bounced"].Int()
			deferred := row[0]["deferred"].Int()

			openedCount, _ := g.DB().Model("mailstat_opened").
				WhereGTE("log_time_millis", startTimeMs).
				WhereLTE("log_time_millis", endTimeMs).
				Fields("count(distinct postfix_message_id) as opened").
				Value()
			clickedCount, _ := g.DB().Model("mailstat_clicked").
				WhereGTE("log_time_millis", startTimeMs).
				WhereLTE("log_time_millis", endTimeMs).
				Fields("count(distinct postfix_message_id) as clicked").
				Value()

			opened := openedCount.Int()
			clicked := clickedCount.Int()

			var deliveryRate, bounceRate, openRate, clickRate float64
			if sends > 0 {
				deliveryRate = public.Round(float64(delivered)/float64(sends)*100, 2)
				bounceRate = public.Round(float64(bounced)/float64(sends)*100, 2)
				openRate = public.Round(float64(opened)/float64(sends)*100, 2)
				clickRate = public.Round(float64(clicked)/float64(sends)*100, 2)
			}

			res.Data.Summary = v1.CampaignSummaryMetric{
				TotalRecipients: sends,
				TotalSent:       sends,
				Delivered:       delivered,
				Bounced:         bounced,
				Deferred:        deferred,
				Opened:          opened,
				Clicked:         clicked,
				DeliveryRate:    deliveryRate,
				BounceRate:      bounceRate,
				OpenRate:        openRate,
				ClickRate:       clickRate,
			}
		}
	}

	// 2. Daily Stats Breakdown
	dailySql := ""
	var dailyArgs []interface{}
	if campaignId > 0 {
		dailySql = cteSql + `
			SELECT 
				to_char(to_timestamp(cr.sent_time), 'YYYY-MM-DD') as day,
				count(*) as sends,
				coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
				coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced,
				coalesce(sum(case when sm.status='deferred' then 1 else 0 end), 0) as deferred
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
			GROUP BY day ORDER BY day ASC
		`
		dailyArgs = nil
	} else {
		dailySql = `
			SELECT 
				to_char(to_timestamp(sm.log_time_millis / 1000), 'YYYY-MM-DD') as day,
				count(*) as sends,
				coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
				coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced,
				coalesce(sum(case when sm.status='deferred' then 1 else 0 end), 0) as deferred
			FROM mailstat_send_mails sm
			WHERE sm.log_time_millis >= ? AND sm.log_time_millis <= ?
		`
		dailyArgs = []interface{}{startTimeMs, endTimeMs}
		if domainFilter != "" {
			dailySql = `
				SELECT 
					to_char(to_timestamp(sm.log_time_millis / 1000), 'YYYY-MM-DD') as day,
					count(*) as sends,
					coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
					coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced,
					coalesce(sum(case when sm.status='deferred' then 1 else 0 end), 0) as deferred
				FROM mailstat_send_mails sm
				JOIN mailstat_senders s ON sm.postfix_message_id = s.postfix_message_id
				WHERE sm.log_time_millis >= ? AND sm.log_time_millis <= ?
				  AND lower(split_part(s.sender, '@', 2)) = ?
			`
			dailyArgs = append(dailyArgs, domainFilter)
		}
		dailySql += " GROUP BY day ORDER BY day ASC"
	}

	var dailyRows []gdb.Record
	var dailyErr error
	if len(dailyArgs) > 0 {
		dailyRows, dailyErr = g.DB().Query(ctx, dailySql, dailyArgs...)
	} else {
		dailyRows, dailyErr = g.DB().Query(ctx, dailySql)
	}
	res.Data.DailyStats = make([]v1.CampaignDailyMetric, 0)
	if dailyErr == nil {
		for _, row := range dailyRows {
			sends := row["sends"].Int()
			delivered := row["delivered"].Int()
			bounced := row["bounced"].Int()
			deferred := row["deferred"].Int()

			var dRate, bRate float64
			if sends > 0 {
				dRate = public.Round(float64(delivered)/float64(sends)*100, 2)
				bRate = public.Round(float64(bounced)/float64(sends)*100, 2)
			}

			res.Data.DailyStats = append(res.Data.DailyStats, v1.CampaignDailyMetric{
				Date:         row["day"].String(),
				Sends:        sends,
				Delivered:    delivered,
				Bounced:      bounced,
				Deferred:     deferred,
				DeliveryRate: dRate,
				BounceRate:   bRate,
			})
		}
	}

	// 3. Bounce Reasons Summary
	var reasonSql string
	var reasonArgs []interface{}
	if campaignId > 0 {
		reasonSql = cteSql + `
			SELECT 
				coalesce(nullif(sm.dsn, ''), '5.0.0') as dsn,
				sm.description as raw_reason,
				count(*) as count
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
			WHERE sm.status = 'bounced'
			GROUP BY sm.dsn, sm.description ORDER BY count DESC LIMIT 30
		`
		reasonArgs = nil
	} else {
		reasonSql = `
			SELECT 
				coalesce(nullif(sm.dsn, ''), '5.0.0') as dsn,
				sm.description as raw_reason,
				count(*) as count
			FROM mailstat_send_mails sm
			WHERE sm.status = 'bounced'
			  AND sm.log_time_millis >= ? AND sm.log_time_millis <= ?
		`
		reasonArgs = []interface{}{startTimeMs, endTimeMs}
		if domainFilter != "" {
			reasonSql = `
				SELECT 
					coalesce(nullif(sm.dsn, ''), '5.0.0') as dsn,
					sm.description as raw_reason,
					count(*) as count
				FROM mailstat_send_mails sm
				JOIN mailstat_senders s ON sm.postfix_message_id = s.postfix_message_id
				WHERE sm.status = 'bounced'
				  AND sm.log_time_millis >= ? AND sm.log_time_millis <= ?
				  AND lower(split_part(s.sender, '@', 2)) = ?
			`
			reasonArgs = append(reasonArgs, domainFilter)
		}
		reasonSql += " GROUP BY sm.dsn, sm.description ORDER BY count DESC LIMIT 30"
	}

	var reasonRows []gdb.Record
	var reasonErr error
	if len(reasonArgs) > 0 {
		reasonRows, reasonErr = g.DB().Query(ctx, reasonSql, reasonArgs...)
	} else {
		reasonRows, reasonErr = g.DB().Query(ctx, reasonSql)
	}
	res.Data.BounceReasonsSummary = make([]v1.BounceReasonSummaryItem, 0)
	totalFailedCount := 0
	if reasonErr == nil {
		for _, row := range reasonRows {
			totalFailedCount += row["count"].Int()
		}
		for _, row := range reasonRows {
			c := row["count"].Int()
			dsn := row["dsn"].String()
			raw := row["raw_reason"].String()
			cat := categorizeBounceReason(dsn, raw)

			pct := float64(0)
			if totalFailedCount > 0 {
				pct = public.Round(float64(c)/float64(totalFailedCount)*100, 2)
			}

			res.Data.BounceReasonsSummary = append(res.Data.BounceReasonsSummary, v1.BounceReasonSummaryItem{
				ReasonCategory: cat,
				Dsn:            dsn,
				Count:          c,
				Percentage:     pct,
				SampleError:    raw,
			})
		}
	}

	// 4. Sending Domains Summary
	res.Data.SendingDomains = make([]v1.DomainMetricItem, 0)
	if campaignId > 0 {
		domSql := cteSql + `
			SELECT 
				cr.sending_domain,
				count(*) as sends,
				coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
				coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
			GROUP BY cr.sending_domain
			ORDER BY sends DESC
			LIMIT 100
		`
		domRows, qErr := g.DB().Query(ctx, domSql)
		if qErr == nil {
			for _, row := range domRows {
				s := row["sends"].Int()
				del := row["delivered"].Int()
				bou := row["bounced"].Int()
				var dRate, bRate float64
				if s > 0 {
					dRate = public.Round(float64(del)/float64(s)*100, 2)
					bRate = public.Round(float64(bou)/float64(s)*100, 2)
				}
				res.Data.SendingDomains = append(res.Data.SendingDomains, v1.DomainMetricItem{
					Domain:       row["sending_domain"].String(),
					Sends:        s,
					Delivered:    del,
					Bounced:      bou,
					DeliveryRate: dRate,
					BounceRate:   bRate,
				})
			}
		}
	}

	// 5. Mail Providers Breakdown
	res.Data.MailProviders = make([]v1.ProviderMetricItem, 0)
	var provSql string
	var provArgs []interface{}
	if campaignId > 0 {
		provSql = cteSql + `
			SELECT 
				sm.mail_provider,
				count(*) as sends,
				coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
				coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
			GROUP BY sm.mail_provider ORDER BY sends DESC LIMIT 15
		`
		provArgs = nil
	} else {
		provSql = `
			SELECT 
				sm.mail_provider,
				count(*) as sends,
				coalesce(sum(case when sm.status='sent' and sm.dsn like '2.%' then 1 else 0 end), 0) as delivered,
				coalesce(sum(case when sm.status='bounced' then 1 else 0 end), 0) as bounced
			FROM mailstat_send_mails sm
			WHERE sm.log_time_millis >= ? AND sm.log_time_millis <= ?
			GROUP BY sm.mail_provider
			ORDER BY sends DESC
			LIMIT 15
		`
		provArgs = []interface{}{startTimeMs, endTimeMs}
	}

	var provRows []gdb.Record
	var provErr error
	if len(provArgs) > 0 {
		provRows, provErr = g.DB().Query(ctx, provSql, provArgs...)
	} else {
		provRows, provErr = g.DB().Query(ctx, provSql)
	}
	if provErr == nil {
		for _, row := range provRows {
			s := row["sends"].Int()
			del := row["delivered"].Int()
			bou := row["bounced"].Int()
			var dRate, bRate float64
			if s > 0 {
				dRate = public.Round(float64(del)/float64(s)*100, 2)
				bRate = public.Round(float64(bou)/float64(s)*100, 2)
			}
			providerName := row["mail_provider"].String()
			if providerName == "" {
				providerName = "other"
			}
			res.Data.MailProviders = append(res.Data.MailProviders, v1.ProviderMetricItem{
				MailProvider: providerName,
				Sends:        s,
				Delivered:    del,
				Bounced:      bou,
				DeliveryRate: dRate,
				BounceRate:   bRate,
			})
		}
	}

	// 6. Recent Failed Emails Sample
	res.Data.RecentFailedEmails = make([]v1.FailedEmailSampleItem, 0)
	var sampleSql string
	var sampleArgs []interface{}
	if campaignId > 0 {
		sampleSql = cteSql + `
			SELECT 
				cr.recipient,
				cr.sending_domain,
				sm.status,
				sm.dsn,
				sm.relay,
				sm.description as reason,
				cr.sent_time as log_time
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
			WHERE sm.status != 'sent'
			ORDER BY cr.sent_time DESC LIMIT ?
		`
		sampleArgs = []interface{}{failedLimit}
	} else {
		sampleSql = `
			SELECT 
				sm.recipient,
				coalesce(s.sender, '') as sender,
				lower(split_part(s.sender, '@', 2)) as sending_domain,
				sm.status,
				sm.dsn,
				sm.relay,
				sm.description as reason,
				(sm.log_time_millis / 1000) as log_time
			FROM mailstat_send_mails sm
			LEFT JOIN mailstat_senders s ON sm.postfix_message_id = s.postfix_message_id
			WHERE sm.status != 'sent'
			  AND sm.log_time_millis >= ? AND sm.log_time_millis <= ?
		`
		sampleArgs = []interface{}{startTimeMs, endTimeMs}
		if domainFilter != "" {
			sampleSql += " AND lower(split_part(s.sender, '@', 2)) = ?"
			sampleArgs = append(sampleArgs, domainFilter)
		}
		sampleSql += " ORDER BY sm.log_time_millis DESC LIMIT ?"
		sampleArgs = append(sampleArgs, failedLimit)
	}

	sampleRows, qErr := g.DB().Query(ctx, sampleSql, sampleArgs...)
	if qErr == nil {
		for _, row := range sampleRows {
			lt := row["log_time"].Int64()
			dateFormatted := ""
			if lt > 0 {
				dateFormatted = time.Unix(lt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
			}
			sender := ""
			if task != nil {
				sender = task.Addresser
			}
			if s, ok := row["sender"]; ok && s.String() != "" {
				sender = s.String()
			}
			res.Data.RecentFailedEmails = append(res.Data.RecentFailedEmails, v1.FailedEmailSampleItem{
				Recipient:     row["recipient"].String(),
				Sender:        sender,
				SendingDomain: row["sending_domain"].String(),
				Status:        row["status"].String(),
				Dsn:           row["dsn"].String(),
				Relay:         row["relay"].String(),
				Reason:        row["reason"].String(),
				LogTime:       lt,
				Date:          dateFormatted,
			})
		}
	}

	res.SetSuccess("Campaign statistics retrieved successfully")
	return res, nil
}

// CampaignsList lists all email campaigns with their delivery & bounce statistics
func (c *ControllerV1) CampaignsList(ctx context.Context, req *v1.CampaignsListReq) (res *v1.CampaignsListRes, err error) {
	if err := authenticateApiRequest(ctx); err != nil {
		res = &v1.CampaignsListRes{}
		res.Code = 401
		res.SetError(err)
		return res, nil
	}

	res = &v1.CampaignsListRes{}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	} else if pageSize > 200 {
		pageSize = 200
	}

	query := g.DB().Model("email_tasks").Safe()
	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch status {
	case "running":
		query = query.Where("pause", 0).Where("task_process", 1)
	case "paused":
		query = query.Where("pause = 1 OR task_process = 3")
	case "completed":
		query = query.Where("task_process", 2)
	case "draft":
		query = query.Where("task_process", 0)
	}

	total, err := query.Count()
	if err != nil {
		res.Code = 500
		res.SetError(err)
		return res, nil
	}

	var tasks []entity.EmailTask
	err = query.Order("id desc").Page(page, pageSize).Scan(&tasks)
	if err != nil {
		res.Code = 500
		res.SetError(err)
		return res, nil
	}

	res.Data.Total = total
	res.Data.Page = page
	res.Data.List = make([]v1.CampaignListItem, 0, len(tasks))

	for _, t := range tasks {
		statusText := "running"
		if t.Pause == 1 {
			statusText = "paused"
		} else if t.TaskProcess == 0 {
			statusText = "draft"
		} else if t.TaskProcess == 2 {
			statusText = "completed"
		} else if t.TaskProcess == 3 {
			statusText = "paused"
		}

		var dRate, bRate float64
		if t.SendsCount > 0 {
			dRate = public.Round(float64(t.DeliveredCount)/float64(t.SendsCount)*100, 2)
			bRate = public.Round(float64(t.BouncedCount)/float64(t.SendsCount)*100, 2)
		}

		startTimeFormatted := ""
		if t.StartTime > 0 {
			startTimeFormatted = time.Unix(int64(t.StartTime), 0).UTC().Format("2006-01-02 15:04:05 UTC")
		}

		res.Data.List = append(res.Data.List, v1.CampaignListItem{
			Id:                   t.Id,
			TaskName:             t.TaskName,
			Subject:              t.Subject,
			Addresser:            t.Addresser,
			RecipientCount:       t.RecipientCount,
			SendsCount:           t.SendsCount,
			DeliveredCount:       t.DeliveredCount,
			BouncedCount:         t.BouncedCount,
			DeferredCount:        t.DeferredCount,
			DeliveryRate:         dRate,
			BounceRate:           bRate,
			TaskProcess:          t.TaskProcess,
			StatusText:           statusText,
			IsPaused:             t.Pause == 1 || t.TaskProcess == 3,
			StartTime:            int64(t.StartTime),
			StartTimeFormatted:   startTimeFormatted,
			CreateTime:           int64(t.CreateTime),
			SendingWindowEnabled: t.SendingWindowEnabled,
			SendingWindowStart:   t.SendingWindowStart,
			SendingWindowEnd:     t.SendingWindowEnd,
			SendingWindowTz:      t.SendingWindowTz,
		})
	}

	res.SetSuccess("Campaigns retrieved successfully")
	return res, nil
}

// CampaignFailedList returns paginated list of failed and bounced email logs
func (c *ControllerV1) CampaignFailedList(ctx context.Context, req *v1.CampaignFailedListReq) (res *v1.CampaignFailedListRes, err error) {
	if err := authenticateApiRequest(ctx); err != nil {
		res = &v1.CampaignFailedListRes{}
		res.Code = 401
		res.SetError(err)
		return res, nil
	}

	res = &v1.CampaignFailedListRes{}

	r := g.RequestFromCtx(ctx)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	} else if pageSize > 200 {
		pageSize = 200
	}

	// Time filters
	var startTime, endTime int64
	if req.StartTime > 0 {
		startTime = req.StartTime
	}
	if req.EndTime > 0 {
		endTime = req.EndTime
	}
	if req.StartDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.StartDate), time.UTC); err == nil {
			startTime = t.Unix()
		}
	}
	if req.EndDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.EndDate), time.UTC); err == nil {
			endTime = t.Unix() + 86399
		}
	}

	var domainFilter string
	if r != nil {
		qDomain := strings.TrimSpace(r.GetQuery("sending_domain").String())
		if qDomain == "" {
			qDomain = strings.TrimSpace(r.GetQuery("domain").String())
		}
		if qDomain != "" {
			domainFilter = strings.ToLower(strings.TrimPrefix(qDomain, "@"))
		}
	}
	if domainFilter == "" && req.SendingDomain != "" && strings.Contains(req.SendingDomain, ".") {
		domainFilter = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(req.SendingDomain), "@"))
	}
	if domainFilter == "" && req.Domain != "" && strings.Contains(req.Domain, ".") {
		domainFilter = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(req.Domain), "@"))
	}
	if domainFilter != "" && !strings.Contains(domainFilter, ".") {
		domainFilter = ""
	}

	dsnFilter := strings.TrimSpace(req.Dsn)

	startTimeMs := startTime * 1000
	endTimeMs := (endTime + 1) * 1000

	var listSql, countSql string
	var countArgs, listArgs []interface{}

	var defaultSender string

	if req.CampaignId > 0 {
		var task entity.EmailTask
		if err := g.DB().Model("email_tasks").Where("id", req.CampaignId).Scan(&task); err == nil && task.Id > 0 {
			defaultSender = task.Addresser
		}

		var timeFilterSql string
		if startTime > 0 && endTime > 0 {
			timeFilterSql = fmt.Sprintf(" AND sent_time >= %d AND sent_time <= %d", startTime, endTime)
		} else if startTime > 0 {
			timeFilterSql = fmt.Sprintf(" AND sent_time >= %d", startTime)
		} else if endTime > 0 {
			timeFilterSql = fmt.Sprintf(" AND sent_time <= %d", endTime)
		}
		var domainFilterSql string
		if domainFilter != "" {
			domainFilterSql = fmt.Sprintf(" AND split_part(message_id, '@', 2) = '%s'", strings.ReplaceAll(domainFilter, "'", "''"))
		}

		cteSql := fmt.Sprintf(`
			WITH cr AS MATERIALIZED (
				SELECT message_id, split_part(message_id, '@', 2) as sending_domain, recipient, sent_time
				FROM recipient_info
				WHERE task_id = %d AND is_sent = 1 AND message_id != ''%s%s
			)
		`, req.CampaignId, timeFilterSql, domainFilterSql)

		countSql = cteSql + `
			SELECT count(*) as count
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
			WHERE sm.status != 'sent'
		`
		listSql = cteSql + `
			SELECT 
				cr.recipient,
				cr.sending_domain,
				sm.status,
				sm.dsn,
				sm.relay,
				sm.description as reason,
				cr.sent_time as log_time
			FROM cr
			JOIN mailstat_message_ids mi ON cr.message_id = mi.message_id
			JOIN mailstat_send_mails sm ON mi.postfix_message_id = sm.postfix_message_id
			WHERE sm.status != 'sent'
		`
		countArgs = []interface{}{}
		listArgs = []interface{}{}
		if dsnFilter != "" {
			countSql += " AND sm.dsn LIKE ?"
			listSql += " AND sm.dsn LIKE ?"
			countArgs = append(countArgs, dsnFilter+"%")
			listArgs = append(listArgs, dsnFilter+"%")
		}
		listSql += " ORDER BY cr.sent_time DESC LIMIT ? OFFSET ?"
		offset := (page - 1) * pageSize
		listArgs = append(listArgs, pageSize, offset)
	} else {
		countSql = `
			SELECT count(*) as count
			FROM mailstat_send_mails sm
			WHERE sm.status != 'sent'
		`
		listSql = `
			SELECT 
				sm.recipient,
				coalesce(s.sender, '') as sender,
				lower(split_part(s.sender, '@', 2)) as sending_domain,
				sm.status,
				sm.dsn,
				sm.relay,
				sm.description as reason,
				(sm.log_time_millis / 1000) as log_time
			FROM mailstat_send_mails sm
			LEFT JOIN mailstat_senders s ON sm.postfix_message_id = s.postfix_message_id
			WHERE sm.status != 'sent'
		`
		countArgs = []interface{}{}
		listArgs = []interface{}{}
		if startTime > 0 {
			countSql += " AND sm.log_time_millis >= ?"
			listSql += " AND sm.log_time_millis >= ?"
			countArgs = append(countArgs, startTimeMs)
			listArgs = append(listArgs, startTimeMs)
		}
		if endTime > 0 {
			countSql += " AND sm.log_time_millis <= ?"
			listSql += " AND sm.log_time_millis <= ?"
			countArgs = append(countArgs, endTimeMs)
			listArgs = append(listArgs, endTimeMs)
		}
		if domainFilter != "" {
			countSql = `
				SELECT count(*) as count
				FROM mailstat_send_mails sm
				JOIN mailstat_senders s ON sm.postfix_message_id = s.postfix_message_id
				WHERE sm.status != 'sent'
			`
			if startTime > 0 {
				countSql += " AND sm.log_time_millis >= ?"
			}
			if endTime > 0 {
				countSql += " AND sm.log_time_millis <= ?"
			}
			countSql += " AND lower(split_part(s.sender, '@', 2)) = ?"
			listSql += " AND lower(split_part(s.sender, '@', 2)) = ?"
			countArgs = append(countArgs, domainFilter)
			listArgs = append(listArgs, domainFilter)
		}
		if dsnFilter != "" {
			countSql += " AND sm.dsn LIKE ?"
			listSql += " AND sm.dsn LIKE ?"
			countArgs = append(countArgs, dsnFilter+"%")
			listArgs = append(listArgs, dsnFilter+"%")
		}
		listSql += " ORDER BY sm.log_time_millis DESC LIMIT ? OFFSET ?"
		offset := (page - 1) * pageSize
		listArgs = append(listArgs, pageSize, offset)
	}

	countRow, qErr := g.DB().Query(ctx, countSql, countArgs...)
	total := 0
	if qErr == nil && len(countRow) > 0 {
		total = countRow[0]["count"].Int()
	}

	rows, qErr := g.DB().Query(ctx, listSql, listArgs...)

	res.Data.Total = total
	res.Data.Page = page
	res.Data.List = make([]v1.FailedEmailSampleItem, 0)

	if qErr == nil {
		for _, row := range rows {
			lt := row["log_time"].Int64()
			dateFormatted := ""
			if lt > 0 {
				dateFormatted = time.Unix(lt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
			}
			sender := defaultSender
			if s, ok := row["sender"]; ok && s.String() != "" {
				sender = s.String()
			}
			res.Data.List = append(res.Data.List, v1.FailedEmailSampleItem{
				Recipient:     row["recipient"].String(),
				Sender:        sender,
				SendingDomain: row["sending_domain"].String(),
				Status:        row["status"].String(),
				Dsn:           row["dsn"].String(),
				Relay:         row["relay"].String(),
				Reason:        row["reason"].String(),
				LogTime:       lt,
				Date:          dateFormatted,
			})
		}
	}

	res.SetSuccess("Failed emails retrieved successfully")
	return res, nil
}
