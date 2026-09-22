package batch_mail

import (
	"billionmail-core/api/batch_mail/v1"
	"billionmail-core/internal/model/entity"
	"billionmail-core/internal/service/contact"
	"billionmail-core/internal/service/public"
	"billionmail-core/internal/service/rbac"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/grand"
	"github.com/gogf/gf/v2/util/guid"
)

// parseRecipients parses and deduplicates recipient email addresses from string and list inputs.
// Supports comma, semicolon, space, newline, and carriage return delimiters.
func parseRecipients(recipientStr string, recipientList []string) []string {
	var rawList []string
	if len(recipientList) > 0 {
		rawList = append(rawList, recipientList...)
	}
	if recipientStr != "" {
		replacer := strings.NewReplacer(";", ",", "\n", ",", "\r", ",", "\t", ",", " ", ",")
		normalized := replacer.Replace(recipientStr)
		parts := strings.Split(normalized, ",")
		rawList = append(rawList, parts...)
	}

	seen := make(map[string]bool)
	var result []string
	for _, item := range rawList {
		email := strings.ToLower(strings.TrimSpace(item))
		if email != "" && strings.Contains(email, "@") && !seen[email] {
			seen[email] = true
			result = append(result, email)
		}
	}
	return result
}

// generateMessageId produces a unique RFC-compliant Message-ID string.
func generateMessageId(addresser string) string {
	domainPart := "billionmail"
	parts := strings.SplitN(addresser, "@", 2)
	if len(parts) > 1 && parts[1] != "" {
		domainPart = parts[1]
	}
	randomBytes := grand.B(16)
	randomID := hex.EncodeToString(randomBytes)
	timestampMillis := time.Now().UnixMilli()
	return fmt.Sprintf("%d.%s@%s", timestampMillis, randomID, domainPart)
}

// getOrCreateApiTemplate gets or initializes an api_templates entity for a specific templateId.
func getOrCreateApiTemplate(ctx context.Context, templateId int, addresser string) (*entity.ApiTemplates, error) {
	var tpl entity.ApiTemplates
	err := g.DB().Model("api_templates").Where("template_id", templateId).Where("active", 1).Order("id asc").Limit(1).Scan(&tpl)
	if err == nil && tpl.Id > 0 {
		return &tpl, nil
	}

	// Verify template exists in email_templates
	emailTpl, err := getEmailTemplateById(ctx, templateId)
	if err != nil {
		return nil, gerror.New(public.LangCtx(ctx, "Email template does not exist"))
	}

	now := int(time.Now().Unix())
	newKey := guid.S() + guid.S()
	if len(newKey) > 64 {
		newKey = newKey[:64]
	}
	addr := addresser
	if addr == "" {
		addr = "all_replies@b2bprosperity.com"
	}
	subj := emailTpl.TempName
	if subj == "" {
		subj = "API Campaign"
	}
	insertRes, err := g.DB().Model("api_templates").Insert(g.Map{
		"api_key":              newKey,
		"api_name":             fmt.Sprintf("API Template %d", templateId),
		"template_id":          templateId,
		"subject":              subj,
		"addresser":            addr,
		"full_name":            "",
		"unsubscribe":          0,
		"track_open":           1,
		"track_click":          1,
		"active":               1,
		"create_time":          now,
		"update_time":          now,
		"expire_time":          0,
		"last_key_update_time": now,
		"group_id":             0,
	})
	if err != nil {
		return nil, gerror.New(public.LangCtx(ctx, "Failed to create API template: {}", err.Error()))
	}
	newId, _ := insertRes.LastInsertId()
	tpl.Id = int(newId)
	tpl.ApiKey = newKey
	tpl.ApiName = fmt.Sprintf("API Template %d", templateId)
	tpl.TemplateId = templateId
	tpl.Subject = subj
	tpl.Addresser = addr
	tpl.Active = 1
	return &tpl, nil
}

// resolveApiTemplate authenticates the request using either the Master Admin API Token
// (Bearer token in Authorization header or x-api-key) or a template-specific API key.
func resolveApiTemplate(ctx context.Context, apiKeyHeader, authHeader string, reqTemplateId int, reqAddresser string, clientIP string) (*entity.ApiTemplates, bool, error) {
	r := g.RequestFromCtx(ctx)

	// 1. Extract token/key from all potential sources
	token := strings.TrimSpace(apiKeyHeader)
	if token == "" {
		token = strings.TrimSpace(authHeader)
	}
	if token == "" && r != nil {
		token = strings.TrimSpace(r.GetHeader("x-api-key"))
		if token == "" {
			token = strings.TrimSpace(r.GetHeader("Authorization"))
		}
		if token == "" {
			token = strings.TrimSpace(r.Get("api_key").String())
		}
		if token == "" {
			token = strings.TrimSpace(r.Get("x-api-key").String())
		}
		if token == "" {
			token = strings.TrimSpace(r.Get("token").String())
		}
	}

	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return nil, false, gerror.New(public.LangCtx(ctx, "API key or Authorization token is required"))
	}

	// 2. Check direct match in api_templates table
	var apiTemplate entity.ApiTemplates
	err := g.DB().Model("api_templates").Where("api_key", token).Where("active", 1).Scan(&apiTemplate)
	if err == nil && apiTemplate.Id > 0 {
		if reqTemplateId > 0 && reqTemplateId != apiTemplate.TemplateId {
			overrideTpl, err := getOrCreateApiTemplate(ctx, reqTemplateId, reqAddresser)
			if err == nil && overrideTpl != nil {
				return overrideTpl, false, nil
			}
		}
		return &apiTemplate, false, nil
	}

	// 3. Check JWT token (Master Admin API Token or Admin Session Token)
	claims, err := rbac.JWT().ParseToken(token)
	if err == nil && claims != nil {
		isAdmin := claims.ApiToken || claims.Username == "admin"
		if !isAdmin {
			for _, role := range claims.Roles {
				if role == "admin" {
					isAdmin = true
					break
				}
			}
		}

		if !isAdmin {
			return nil, false, gerror.New(public.LangCtx(ctx, "Unauthorized: token does not have admin permissions"))
		}

		// Valid Master Admin API Token!
		if reqTemplateId > 0 {
			tpl, err := getOrCreateApiTemplate(ctx, reqTemplateId, reqAddresser)
			if err != nil {
				return nil, true, err
			}
			return tpl, true, nil
		}

		// No template_id specified; pick primary active api_template
		var defaultTpl entity.ApiTemplates
		err = g.DB().Model("api_templates").Where("active", 1).Order("id asc").Limit(1).Scan(&defaultTpl)
		if err == nil && defaultTpl.Id > 0 {
			return &defaultTpl, true, nil
		}

		// Fallback to first available email_template
		var emailTpl entity.EmailTemplate
		err = g.DB().Model("email_templates").Order("id asc").Limit(1).Scan(&emailTpl)
		if err != nil || emailTpl.Id == 0 {
			return nil, true, gerror.New(public.LangCtx(ctx, "No email templates found in system"))
		}

		tpl, err := getOrCreateApiTemplate(ctx, emailTpl.Id, reqAddresser)
		if err != nil {
			return nil, true, err
		}
		return tpl, true, nil
	}

	return nil, false, gerror.New(public.LangCtx(ctx, "API key or token is invalid"))
}

// getApiTemplateByKey provides backward compatibility for looking up an API template
func getApiTemplateByKey(ctx context.Context, apiKey string, clientIP string) (*entity.ApiTemplates, error) {
	tpl, _, err := resolveApiTemplate(ctx, apiKey, "", 0, "", clientIP)
	return tpl, err
}

func (c *ControllerV1) ApiMailSend(ctx context.Context, req *v1.ApiMailSendReq) (res *v1.ApiMailSendRes, err error) {
	res = &v1.ApiMailSendRes{}
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

	// 4. Parse and validate recipient(s) (supports single, comma/semicolon/newline-delimited, and array)
	recipients := parseRecipients(req.Recipient, req.Recipients)
	if len(recipients) == 0 {
		res.Code = 1003
		res.SetError(gerror.New(public.LangCtx(ctx, "Invalid or empty recipient email address(es)")))
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

	// 7. Queue emails for all recipients
	now := int(time.Now().Unix())
	batchData := make([]g.Map, 0, len(recipients))
	for _, recipient := range recipients {
		// Contact and group handling
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
		res.SetError(gerror.New(public.LangCtx(ctx, "No valid recipients to queue")))
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

	if len(batchData) == 1 {
		res.SetSuccess(public.LangCtx(ctx, "Email queued successfully"))
	} else {
		res.SetSuccess(fmt.Sprintf("%d emails queued successfully", len(batchData)))
	}
	return res, nil
}

// check API template by key and client IP
func CheckClientIP(ctx context.Context, Id int, clientIP string) error {

	ipcount, err := g.DB().Model("api_ip_whitelist").
		Where("api_id", Id).Count()
	if err == nil && ipcount > 0 {

		count, err := g.DB().Model("api_ip_whitelist").
			Where("api_id", Id).
			Where("ip", clientIP).
			Count()
		if err != nil {
			return err
		}
		if count == 0 {
			return gerror.New(public.LangCtx(ctx, "IP not allowed"))
		}
	}

	return nil
}

// get email template
func getEmailTemplateById(ctx context.Context, templateId int) (*entity.EmailTemplate, error) {
	var emailTemplate entity.EmailTemplate
	err := g.DB().Model("email_templates").Where("id", templateId).Scan(&emailTemplate)
	if err != nil || emailTemplate.Id == 0 {
		return nil, gerror.New(public.LangCtx(ctx, "Email template does not exist"))
	}
	return &emailTemplate, nil
}

// ensure contact and group exists
func ensureContactAndGroup(ctx context.Context, email string, apiId int) (entity.Contact, error) {
	var contact entity.Contact
	now := int(time.Now().Unix())

	apiGroupName := fmt.Sprintf("api_group_%d", apiId)
	var group entity.ContactGroup
	err := g.DB().Model("bm_contact_groups").Where("name", apiGroupName).Scan(&group)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			group = entity.ContactGroup{}
		} else {
			return contact, fmt.Errorf("failed to query group: %w", err)
		}
	}
	if group.Id == 0 {
		groupResult, err := g.DB().Model("bm_contact_groups").Insert(g.Map{
			"name":        apiGroupName,
			"description": fmt.Sprintf(public.LangCtx(ctx, "API %d automatically created contact group"), apiId),
			"create_time": now,
			"update_time": now,
		})
		if err != nil {
			return contact, err
		}
		groupId, _ := groupResult.LastInsertId()
		group.Id = int(groupId)
	} else {

		count, err := g.DB().Model("bm_contacts").
			Where("email", email).
			Where("group_id", group.Id).
			Where("active", 0).
			Count()
		if err != nil {
			return contact, fmt.Errorf("failed to check unsubscribe: %w", err)
		}
		if count > 0 {
			return contact, fmt.Errorf("the recipient has unsubscribed from the current group and cannot send")
		}
	}

	contactData := g.Map{
		"email":       email,
		"group_id":    group.Id,
		"active":      1,
		"status":      1,
		"create_time": now,
	}
	contactResult, err := g.DB().Model("bm_contacts").
		Data(contactData).
		OnConflict("email,group_id").
		Save()
	if err != nil {
		return contact, err
	}

	contactId, _ := contactResult.LastInsertId()
	contact.Id = int(contactId)
	contact.Email = email
	contact.GroupId = group.Id
	contact.Active = 1
	contact.Status = 1
	contact.CreateTime = now

	return contact, nil
}

//// process mail content and subject
//func processMailContentAndSubject(ctx context.Context, content, subject string, apiTemplate *entity.ApiTemplates, contact entity.Contact, req *v1.ApiMailSendReq) (string, string) {
//	// unsubscribe link processing
//	if apiTemplate.Unsubscribe == 1 {
//		if !strings.Contains(content, "__UNSUBSCRIBE_URL__") && !strings.Contains(content, "{{ UnsubscribeURL . }}") {
//			content = public.AddUnsubscribeButton(content)
//		}
//		//domain := domains.GetBaseURLBySender(req.Addresser)
//		domain := domains.GetBaseURL()
//		unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe", domain)
//		groupURL := fmt.Sprintf("%s/api/unsubscribe/user_group", domain)
//		jwtToken, _ := batch_mail.GenerateUnsubscribeJWT(req.Recipient, apiTemplate.TemplateId, apiTemplate.Id, contact.GroupId)
//		unsubscribeJumpURL := fmt.Sprintf("%s/unsubscribe.html?jwt=%s&email=%s&url_type=%s&url_unsubscribe=%s", domain, jwtToken, req.Recipient, groupURL, unsubscribeURL)
//
//		if contact.Id > 0 {
//			engine := batch_mail.GetTemplateEngine()
//			renderedContent, err := engine.RenderEmailTemplate(ctx, content, &contact, nil, unsubscribeJumpURL)
//			if err == nil {
//				content = renderedContent
//			}
//			renderedSubject, err := engine.RenderEmailTemplate(ctx, subject, &contact, nil, unsubscribeJumpURL)
//			if err == nil {
//				subject = renderedSubject
//			}
//		} else {
//			content = strings.ReplaceAll(content, "{{ UnsubscribeURL . }}", unsubscribeJumpURL)
//		}
//	} else if contact.Id > 0 {
//		engine := batch_mail.GetTemplateEngine()
//		renderedContent, err := engine.RenderEmailTemplate(ctx, content, &contact, nil, "")
//		if err == nil {
//			content = renderedContent
//		}
//		renderedSubject, err := engine.RenderEmailTemplate(ctx, subject, &contact, nil, "")
//		if err == nil {
//			subject = renderedSubject
//		}
//	}
//	return content, subject
//}
//
//// send email
//func sendApiMail(ctx context.Context, apiTemplate *entity.ApiTemplates, subject, content, recipient, addresser string) error {
//
//	// create email sender
//	sender, err := mail_service.NewEmailSenderWithLocal(addresser)
//	if err != nil {
//		return gerror.New(public.LangCtx(ctx, "Failed to create email sender: {}", err))
//	}
//	defer sender.Close()
//
//	// generate message ID
//	messageId := sender.GenerateMessageID()
//	// add 1 billion to prevent conflict with marketing task id
//	//baseURL := domains.GetBaseURLBySender(addresser)
//	baseURL := domains.GetBaseURL()
//	apiTemplate_id := apiTemplate.Id + 1000000000
//	mailTracker := maillog_stat.NewMailTracker(content, apiTemplate_id, messageId, recipient, baseURL)
//	mailTracker.TrackLinks()
//	mailTracker.AppendTrackingPixel()
//	content = mailTracker.GetHTML()
//
//	// create email message
//	message := mail_service.NewMessage(subject, content)
//	message.SetMessageID(messageId)
//	// set sender display name
//	if apiTemplate.FullName != "" {
//		message.SetRealName(apiTemplate.FullName)
//	}
//	// send email
//	err = sender.Send(message, []string{recipient})
//	if err != nil {
//		return err
//	}
//	// record email log
//	messageId = strings.Trim(messageId, "<>")
//	_, err = g.DB().Model("api_mail_logs").Insert(g.Map{
//		"api_id":     apiTemplate.Id,
//		"recipient":  recipient,
//		"message_id": messageId,
//		"addresser":  addresser,
//	})
//	if err != nil {
//		return err
//	}
//	return nil
//}
