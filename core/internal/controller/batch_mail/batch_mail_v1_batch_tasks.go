package batch_mail

import (
	"context"

	"billionmail-core/api/batch_mail/v1"
	"billionmail-core/internal/consts"
	"billionmail-core/internal/service/batch_mail"
	"billionmail-core/internal/service/public"

	"github.com/gogf/gf/v2/errors/gerror"
)

// TaskBatchMove moves multiple tasks to a target workspace
func (c *ControllerV1) TaskBatchMove(ctx context.Context, req *v1.TaskBatchMoveReq) (res *v1.TaskBatchMoveRes, err error) {
	res = &v1.TaskBatchMoveRes{}
	if len(req.TaskIds) == 0 {
		res.Code = 400
		res.SetError(gerror.New(public.LangCtx(ctx, "No tasks selected")))
		return
	}
	err = batch_mail.BatchMoveTasks(ctx, req.TaskIds, req.WorkspaceId)
	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to move tasks: {}", err.Error())))
		return
	}
	_ = public.WriteLog(ctx, public.LogParams{
		Type: consts.LOGTYPE.Task,
		Log:  "Batch moved tasks to workspace",
		Data: req,
	})
	res.SetSuccess(public.LangCtx(ctx, "Tasks moved successfully"))
	return
}

// TaskBatchDelete deletes multiple tasks
func (c *ControllerV1) TaskBatchDelete(ctx context.Context, req *v1.TaskBatchDeleteReq) (res *v1.TaskBatchDeleteRes, err error) {
	res = &v1.TaskBatchDeleteRes{}
	if len(req.TaskIds) == 0 {
		res.Code = 400
		res.SetError(gerror.New(public.LangCtx(ctx, "No tasks selected")))
		return
	}
	err = batch_mail.BatchDeleteTasks(ctx, req.TaskIds)
	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to delete tasks: {}", err.Error())))
		return
	}
	_ = public.WriteLog(ctx, public.LogParams{
		Type: consts.LOGTYPE.Task,
		Log:  "Batch deleted tasks",
		Data: req,
	})
	res.SetSuccess(public.LangCtx(ctx, "Tasks deleted successfully"))
	return
}
