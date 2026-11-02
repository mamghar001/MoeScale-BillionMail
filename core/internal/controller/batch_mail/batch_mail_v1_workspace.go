package batch_mail

import (
	"context"

	"billionmail-core/api/batch_mail/v1"
	"billionmail-core/internal/consts"
	"billionmail-core/internal/service/batch_mail"
	"billionmail-core/internal/service/public"

	"github.com/gogf/gf/v2/errors/gerror"
)

// WorkspaceList retrieves all workspaces with live campaign counts
func (c *ControllerV1) WorkspaceList(ctx context.Context, req *v1.WorkspaceListReq) (res *v1.WorkspaceListRes, err error) {
	res = &v1.WorkspaceListRes{}
	workspaces, err := batch_mail.GetWorkspaces(ctx)
	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to get workspace list: {}", err.Error())))
		return
	}
	res.Data = workspaces
	res.SetSuccess(public.LangCtx(ctx, "Workspace list retrieved successfully"))
	return
}

// WorkspaceCreate creates a new workspace
func (c *ControllerV1) WorkspaceCreate(ctx context.Context, req *v1.WorkspaceCreateReq) (res *v1.WorkspaceCreateRes, err error) {
	res = &v1.WorkspaceCreateRes{}
	id, err := batch_mail.CreateWorkspace(ctx, req.Name, req.Color)
	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to create workspace: {}", err.Error())))
		return
	}
	res.Data.Id = id
	_ = public.WriteLog(ctx, public.LogParams{
		Type: consts.LOGTYPE.Task,
		Log:  "Created workspace: " + req.Name,
		Data: req,
	})
	res.SetSuccess(public.LangCtx(ctx, "Workspace created successfully"))
	return
}

// WorkspaceUpdate updates an existing workspace's name and color
func (c *ControllerV1) WorkspaceUpdate(ctx context.Context, req *v1.WorkspaceUpdateReq) (res *v1.WorkspaceUpdateRes, err error) {
	res = &v1.WorkspaceUpdateRes{}
	err = batch_mail.UpdateWorkspace(ctx, req.Id, req.Name, req.Color)
	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to update workspace: {}", err.Error())))
		return
	}
	_ = public.WriteLog(ctx, public.LogParams{
		Type: consts.LOGTYPE.Task,
		Log:  "Updated workspace: " + req.Name,
		Data: req,
	})
	res.SetSuccess(public.LangCtx(ctx, "Workspace updated successfully"))
	return
}

// WorkspaceDelete deletes a workspace and moves its campaigns to default/unassigned (0)
func (c *ControllerV1) WorkspaceDelete(ctx context.Context, req *v1.WorkspaceDeleteReq) (res *v1.WorkspaceDeleteRes, err error) {
	res = &v1.WorkspaceDeleteRes{}
	err = batch_mail.DeleteWorkspace(ctx, req.Id)
	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to delete workspace: {}", err.Error())))
		return
	}
	_ = public.WriteLog(ctx, public.LogParams{
		Type: consts.LOGTYPE.Task,
		Log:  "Deleted workspace ID",
		Data: req,
	})
	res.SetSuccess(public.LangCtx(ctx, "Workspace deleted successfully"))
	return
}
