package batch_mail

import (
	"billionmail-core/internal/service/contact"
	"billionmail-core/internal/service/public"
	"context"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	"billionmail-core/api/batch_mail/v1"
)

func (c *ControllerV1) ApiMailBatchSend(ctx context.Context, req *v1.ApiMailBatchSendReq) (res *v1.ApiMailBatchSendRes, err error) {
	res = &v1.ApiMailBatchSendRes{}
	clientIP := g.RequestFromCtx(ctx).GetClientIp()

	// 1. Resolve API Template & Authenticate (Master Admin API Token or Template API Key)
	apiTemplate, isAdminToken, err := resolveApiTemplate(ctx, req.ApiKey, req.Authorization, req.TemplateId, req.Addresser, clientIP)
	if err != nil {
		res.Code = 1001
		res.SetError(gerror.New(public.LangCtx(ctx, err.Error())))
		return res, nil
	}

	// 2. Check client IP (only enforce if not Master Admin token)
	if !isAdminToken {
		err = CheckClientIP(ctx, apiTemplate.Id, clientIP)
		if err != nil {
			res.Code = 1002
			res.SetError(gerror.New(public.LangCtx(ctx, err.Error())))
			return res, nil
		}
	}

	// 3. Check email template
	_, err = getEmailTemplateById(ctx, apiTemplate.TemplateId)
	if err != nil {
		res.Code = 1004
		res.SetError(gerror.New(public.LangCtx(ctx, "Email template does not exist")))
		return res, nil
	}

	// 4. Parse and validate recipient(s) (supports array, comma/semicolon/newline-delimited string, or both)
	validRecipients := parseRecipients(req.Recipient, req.Recipients)
	if len(validRecipients) == 0 {
		res.Code = 1003
		res.SetError(gerror.New(public.LangCtx(ctx, "Recipients cannot be empty")))
		return res, nil
	}

	// 5. Process addresser
	addresser := req.Addresser
	if addresser == "" {
		addresser = apiTemplate.Addresser
	}
	if addresser == "" {
		addresser = "all_replies@b2bprosperity.com"
	}

	// 6. Process attributes & custom subject
	attribs := req.Attribs
	if attribs == nil {
		attribs = make(map[string]string)
	}
	if req.Subject != "" {
		attribs["subject"] = req.Subject
	}

	batchData := make([]g.Map, 0, len(validRecipients))
	now := int(time.Now().Unix())
	for _, recipient := range validRecipients {
		if apiTemplate.GroupId > 0 {
			_, err = contact.AddContactToGroup(ctx, recipient, apiTemplate.GroupId)
			if err != nil {
				g.Log().Warningf(ctx, "Failed to add contact %s to group %d: %v", recipient, apiTemplate.GroupId, err)
			}
		} else {
			_, err = ensureContactAndGroup(ctx, recipient, apiTemplate.Id)
			if err != nil {
				g.Log().Warningf(ctx, "Failed to ensure contact and group for %s with API ID %d: %v", recipient, apiTemplate.Id, err)
			}
		}

		messageId := generateMessageId(addresser)
		batchData = append(batchData, g.Map{
			"api_id":        apiTemplate.Id,
			"recipient":     recipient,
			"message_id":    messageId,
			"addresser":     addresser,
			"status":        0, // Pending send
			"error_message": "",
			"send_time":     0,
			"create_time":   now,
			"attribs":       attribs,
		})
	}

	if len(batchData) == 0 {
		res.Code = 1003
		res.SetError(gerror.New(public.LangCtx(ctx, "No valid recipients to insert")))
		return res, nil
	}

	batchSize := 1000
	for i := 0; i < len(batchData); i += batchSize {
		end := i + batchSize
		if end > len(batchData) {
			end = len(batchData)
		}
		_, err = g.DB().Model("api_mail_logs").Batch(batchSize).Insert(batchData[i:end])
		if err != nil {
			res.Code = 1005
			res.SetError(gerror.New(public.LangCtx(ctx, "Failed to record email log: {}", err.Error())))
			return res, nil
		}
	}

	res.SetSuccess(public.LangCtx(ctx, "Batch email send request accepted, {} emails queued", len(batchData)))
	return res, nil
}
