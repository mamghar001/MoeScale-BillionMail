package batch_mail

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"billionmail-core/api/batch_mail/v1"
	"billionmail-core/internal/model/entity"
	"billionmail-core/internal/service/batch_mail"
	"billionmail-core/internal/service/domains"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) TaskSamplePreview(ctx context.Context, req *v1.TaskSamplePreviewReq) (res *v1.TaskSamplePreviewRes, err error) {
	res = &v1.TaskSamplePreviewRes{}

	if req.Id <= 0 {
		return nil, gerror.New("task id is required")
	}

	// 1. Fetch Task
	var task entity.EmailTask
	err = g.DB().Model("email_tasks").Where("id", req.Id).Scan(&task)
	if err != nil || task.Id == 0 {
		return nil, gerror.Newf("task %d not found: %v", req.Id, err)
	}

	// 2. Fetch Template
	var template entity.EmailTemplate
	err = g.DB().Model("email_templates").Where("id", task.TemplateId).Scan(&template)
	if err != nil || template.Id == 0 {
		return nil, gerror.Newf("template %d for task not found: %v", task.TemplateId, err)
	}

	// 3. Select Sender (simulate rotation or single sender)
	senderEmail := task.Addresser
	senderName := strings.TrimSpace(task.FullName)

	if task.RotateSenders == 1 {
		var mailboxes []struct {
			Username string `json:"username"`
			FullName string `json:"full_name"`
		}
		_ = g.DB().Model("mailbox").
			Where("active", 1).
			Where("domain != ?", "b2bprosperity.com").
			Scan(&mailboxes)

		if len(mailboxes) > 0 {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			idx := r.Intn(len(mailboxes))
			senderEmail = mailboxes[idx].Username
			if strings.TrimSpace(mailboxes[idx].FullName) != "" {
				senderName = strings.TrimSpace(mailboxes[idx].FullName)
			}
		}
	} else if senderName == "" {
		var mb struct {
			FullName string `json:"full_name"`
		}
		_ = g.DB().Model("mailbox").Where("username", senderEmail).Scan(&mb)
		if strings.TrimSpace(mb.FullName) != "" {
			senderName = strings.TrimSpace(mb.FullName)
		}
	}

	effectiveTask := task
	effectiveTask.Addresser = senderEmail
	effectiveTask.FullName = senderName

	// 4. Select Sample Prospect / Contact
	var contact entity.Contact
	var sampleEmail string

	// Try from recipient_info first (real prospect queued for this task)
	var recipientRow struct {
		Recipient string `json:"recipient"`
	}
	_ = g.DB().Model("recipient_info").
		Where("task_id", task.Id).
		Order("RANDOM()").
		Limit(1).
		Scan(&recipientRow)

	if recipientRow.Recipient != "" {
		sampleEmail = recipientRow.Recipient
		_ = g.DB().Model("bm_contacts").
			Where("email", sampleEmail).
			OrderDesc("create_time").
			Limit(1).
			Scan(&contact)
	}

	// If no recipient in recipient_info, try from bm_contacts by task group
	if contact.Id == 0 && task.GroupId > 0 {
		_ = g.DB().Model("bm_contacts").
			Where("group_id", task.GroupId).
			Order("RANDOM()").
			Limit(1).
			Scan(&contact)
		if contact.Email != "" {
			sampleEmail = contact.Email
		}
	}

	// Fallback realistic mock contact if database has no contact data yet
	if contact.Email == "" {
		mockFirstNames := []string{"David", "Sarah", "Michael", "Emily", "James", "Jessica"}
		mockLastNames := []string{"Miller", "Johnson", "Williams", "Brown", "Davis", "Wilson"}
		mockCompanies := []string{"Apex Solutions", "BlueSky Capital", "NextWave Partners", "Summit Media", "Vanguard Logistics"}
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		fn := mockFirstNames[r.Intn(len(mockFirstNames))]
		ln := mockLastNames[r.Intn(len(mockLastNames))]
		comp := mockCompanies[r.Intn(len(mockCompanies))]

		contact = entity.Contact{
			Email: strings.ToLower(fn) + "." + strings.ToLower(ln) + "@example.com",
			Attribs: map[string]string{
				"first_name": fn,
				"last_name":  ln,
				"company":    comp,
			},
		}
		sampleEmail = contact.Email
	} else if contact.Attribs == nil {
		contact.Attribs = make(map[string]string)
	}

	// 5. Render Template & Subject
	engine := batch_mail.GetTemplateEngine()
	domain := domains.GetBaseURL()
	unsubscribeJumpURL := fmt.Sprintf("%s/unsubscribe.html?sample=1&email=%s", domain, sampleEmail)
	if task.Unsubscribe != 1 {
		unsubscribeJumpURL = ""
	}

	renderedContent, err := engine.RenderEmailTemplate(ctx, template.Content, &contact, &effectiveTask, unsubscribeJumpURL)
	if err != nil {
		g.Log().Warning(ctx, "sample preview render content failed: %v", err)
		renderedContent = template.Content
	}

	renderedSubject, err := engine.RenderEmailTemplate(ctx, task.Subject, &contact, &effectiveTask, unsubscribeJumpURL)
	if err != nil {
		g.Log().Warning(ctx, "sample preview render subject failed: %v", err)
		renderedSubject = task.Subject
	}

	// Also ensure spintax on subject is resolved
	spintaxParser := batch_mail.GetSpintaxParser()
	renderedSubject = spintaxParser.ParseSpintax(renderedSubject)

	// Build recipient display name from attributes
	recipientName := sampleEmail
	if fn, ok := contact.Attribs["first_name"]; ok && fn != "" {
		recipientName = fn
		if ln, ok2 := contact.Attribs["last_name"]; ok2 && ln != "" {
			recipientName += " " + ln
		}
	}

	attribsMap := make(map[string]interface{})
	for k, v := range contact.Attribs {
		attribsMap[k] = v
	}

	res.SetSuccess("ok")
	res.Data = v1.TaskSamplePreviewData{
		TaskId:           task.Id,
		TaskName:         task.TaskName,
		TemplateId:       template.Id,
		TemplateName:     template.TempName,
		SenderEmail:      senderEmail,
		SenderName:       senderName,
		RotateSenders:    task.RotateSenders,
		RecipientEmail:   sampleEmail,
		RecipientName:    recipientName,
		RecipientAttribs: attribsMap,
		RawSubject:       task.Subject,
		RenderedSubject:  renderedSubject,
		RawContent:       template.Content,
		RenderedContent:  renderedContent,
	}

	return res, nil
}
