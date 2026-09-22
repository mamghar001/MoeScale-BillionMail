package batch_mail

import (
	"billionmail-core/internal/consts"
	"billionmail-core/internal/model/entity"
	"billionmail-core/internal/service/batch_mail"
	"billionmail-core/internal/service/domains"
	"billionmail-core/internal/service/mail_service"
	"database/sql"
	"strings"

	"billionmail-core/internal/service/public"
	"context"
	"fmt"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	"billionmail-core/api/batch_mail/v1"
)

func (c *ControllerV1) SendTestEmail(ctx context.Context, req *v1.SendTestEmailReq) (res *v1.SendTestEmailRes, err error) {

	res = &v1.SendTestEmailRes{}
	sender, err := mail_service.NewEmailSenderWithLocal(req.Addresser)
	if err != nil {
		res.Code = 400
		res.SetError(gerror.New(public.LangCtx(ctx, "create email sender failed: : {}", err.Error())))
		return
	}
	defer sender.Close()

	// Parse and clean multiple recipients (comma, semicolon, space, newline separated)
	cleaned := strings.NewReplacer(";", ",", "\n", ",", "\r", ",", "\t", ",", " ", ",").Replace(req.Recipient)
	var recipients []string
	seen := make(map[string]bool)
	for _, p := range strings.Split(cleaned, ",") {
		addr := strings.TrimSpace(p)
		if addr != "" && strings.Contains(addr, "@") {
			if !seen[addr] {
				seen[addr] = true
				recipients = append(recipients, addr)
			}
		}
	}

	if len(recipients) == 0 {
		res.Code = 400
		res.SetError(gerror.New(public.LangCtx(ctx, "no valid recipient email address found")))
		return
	}

	// Resolve sender display name (use FullName if provided, otherwise fetch from mailbox)
	senderName := strings.TrimSpace(req.FullName)
	if senderName == "" {
		var mb entity.Mailbox
		_ = g.DB().Model("mailbox").Where("username", req.Addresser).Scan(&mb)
		senderName = mb.FullName
	}

	var template entity.EmailTemplate
	err = g.DB().Model("email_templates").
		Where("id", req.TemplateId).
		Scan(&template)

	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "failed to get template {}", err)))
		return
	}

	// Base content: if unsubscribe is NOT enabled, strip any unsubscribe button div and placeholders
	baseContent := template.Content
	if req.Unsubscribe != 1 {
		baseContent = strings.ReplaceAll(baseContent, `<div style="padding: 16px 0; text-align: center"><a href="{{ UnsubscribeURL . }}" style="color: #ccc; font-size: 12px">Unsubscribe</a></div>`, "")
		baseContent = strings.ReplaceAll(baseContent, "__UNSUBSCRIBE_URL__", "")
	}

	engine := batch_mail.GetTemplateEngine()
	var successRecipients []string
	var failedRecipients []string

	for _, recipient := range recipients {
		var contact entity.Contact
		cErr := g.DB().Model("bm_contacts").Where("email", recipient).Scan(&contact)
		if cErr != nil && cErr != sql.ErrNoRows {
			g.Log().Error(ctx, "Failed to get contact: %v", cErr)
		}
		if contact.Email == "" {
			contact.Email = recipient
		}

		var unsubscribeJumpURL string
		contentToRender := baseContent

		if req.Unsubscribe == 1 {
			jwtToken, _ := batch_mail.GenerateUnsubscribeJWT(recipient, req.TemplateId, 0, 0)
			domain := domains.GetBaseURL()
			unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe", domain)
			groupURL := fmt.Sprintf("%s/api/unsubscribe/user_group", domain)
			unsubscribeJumpURL = fmt.Sprintf("%s/unsubscribe.html?jwt=%s&email=%s&url_type=%s&url_unsubscribe=%s",
				domain, jwtToken, recipient, groupURL, unsubscribeURL)

			if !strings.Contains(contentToRender, "{{ UnsubscribeURL . }}") {
				contentToRender = public.AddUnsubscribeButton(contentToRender)
			}
			contentToRender = strings.ReplaceAll(contentToRender, "__UNSUBSCRIBE_URL__", "{{ UnsubscribeURL . }}")
		}

		// Always render template for clean variables even without contact in bm_contacts
		renderedContent, rErr := engine.RenderEmailTemplate(ctx, contentToRender, &contact, nil, unsubscribeJumpURL)
		if rErr != nil {
			g.Log().Warning(ctx, "failed to render email content: %v", rErr)
			renderedContent = contentToRender
		}
		if req.Unsubscribe != 1 {
			renderedContent = strings.ReplaceAll(renderedContent, "{{ UnsubscribeURL . }}", "")
			renderedContent = strings.ReplaceAll(renderedContent, "__UNSUBSCRIBE_URL__", "")
		}

		renderedSubject, rErr := engine.RenderEmailTemplate(ctx, req.Subject, &contact, nil, unsubscribeJumpURL)
		if rErr != nil {
			g.Log().Warning(ctx, "failed to render email subject: %v", rErr)
			renderedSubject = req.Subject
		}

		message := mail_service.NewMessage(renderedSubject, renderedContent)
		message.SetMessageID(sender.GenerateMessageID())
		if senderName != "" {
			message.SetRealName(senderName)
		}

		// Send email individually to this recipient (matches campaign envelope and To: header)
		sendErr := sender.Send(message, []string{recipient})
		if sendErr != nil {
			g.Log().Error(ctx, "send email to %s failed: %v", recipient, sendErr)
			failedRecipients = append(failedRecipients, fmt.Sprintf("%s (%v)", recipient, sendErr))
		} else {
			successRecipients = append(successRecipients, recipient)
		}
	}

	if len(successRecipients) == 0 && len(failedRecipients) > 0 {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "send test email failed: {}", strings.Join(failedRecipients, ", "))))
		return
	}

	logMsg := fmt.Sprintf("Send test email: %s to %s successfully", req.Subject, strings.Join(successRecipients, ", "))
	if len(failedRecipients) > 0 {
		logMsg += fmt.Sprintf(" (failed: %s)", strings.Join(failedRecipients, ", "))
	}
	_ = public.WriteLog(ctx, public.LogParams{
		Type: consts.LOGTYPE.Task,
		Log:  logMsg,
	})

	successMsg := fmt.Sprintf("send email successfully (%d recipient(s))", len(successRecipients))
	if len(failedRecipients) > 0 {
		successMsg += fmt.Sprintf(", failed for: %s", strings.Join(failedRecipients, ", "))
	}
	res.SetSuccess(public.LangCtx(ctx, successMsg))
	return res, nil
}
