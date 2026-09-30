package contact

import (
	"billionmail-core/api/contact/v1"
	"billionmail-core/internal/consts"
	"billionmail-core/internal/service/contact_activity"
	"billionmail-core/internal/service/public"
	"billionmail-core/internal/service/rbac"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

func parseUnsubEmails(reqEmail string, reqEmails interface{}) []string {
	seen := make(map[string]bool)
	var result []string

	add := func(raw string) {
		trimmed := strings.ToLower(strings.TrimSpace(raw))
		if trimmed != "" && strings.Contains(trimmed, "@") && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}

	if reqEmail != "" {
		for _, part := range strings.FieldsFunc(reqEmail, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
		}) {
			add(part)
		}
	}

	if reqEmails != nil {
		switch v := reqEmails.(type) {
		case string:
			for _, part := range strings.FieldsFunc(v, func(r rune) bool {
				return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
			}) {
				add(part)
			}
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					add(str)
				}
			}
		case []string:
			for _, str := range v {
				add(str)
			}
		}
	}

	return result
}

func authenticateUnsubRequest(ctx context.Context, authHeader, reqApiKey string) bool {
	r := g.RequestFromCtx(ctx)

	// 1. Check existing Web UI session
	if r != nil && r.Session != nil {
		if r.Session.MustGet("safe_path_pass", false).Bool() {
			if _, err := rbac.JWT().ParseToken(r.Session.MustGet("SignedToken", "").String()); err == nil {
				return true
			}
		}
	}

	// 2. Extract token/key
	token := strings.TrimSpace(reqApiKey)
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
		return false
	}

	// 3. Match api_templates table
	var count int
	count, _ = g.DB().Model("api_templates").Where("api_key", token).Where("active", 1).Count()
	if count > 0 {
		return true
	}

	// 4. Match JWT (Master Token / Admin Token)
	claims, err := rbac.JWT().ParseToken(token)
	if err == nil && claims != nil {
		return claims.ApiToken || claims.Username == "admin" || claims.AccountId > 0
	}

	return false
}

func (c *ControllerV1) UnsubscribeContact(ctx context.Context, req *v1.UnsubscribeContactReq) (res *v1.UnsubscribeContactRes, err error) {
	res = &v1.UnsubscribeContactRes{}

	// Authentication check
	if !authenticateUnsubRequest(ctx, req.Authorization, req.ApiKey) {
		res.Code = 401
		res.SetError(gerror.New(public.LangCtx(ctx, "Unauthorized: invalid API token or credentials")))
		return res, nil
	}

	emails := parseUnsubEmails(req.Email, req.Emails)
	if len(emails) == 0 {
		res.Code = 400
		res.SetError(gerror.New(public.LangCtx(ctx, "Please provide at least one valid email address to unsubscribe")))
		return res, nil
	}

	var totalAffected int64
	now := time.Now().Unix()

	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		for _, email := range emails {
			// Find group IDs where this email exists
			var contacts []struct {
				GroupId int `json:"group_id"`
			}
			_ = tx.Model("bm_contacts").
				Fields("group_id").
				Where("email", email).
				Scan(&contacts)

			var groupIds []int
			for _, item := range contacts {
				if item.GroupId > 0 {
					groupIds = append(groupIds, item.GroupId)
				}
			}

			// Update all contact records to active = 0
			result, err := tx.Model("bm_contacts").
				Where("email", email).
				Data(g.Map{"active": 0}).
				Update()
			if err != nil {
				return err
			}

			affected, _ := result.RowsAffected()
			totalAffected += affected

			// Record unsubscription in unsubscribe_records for each group
			if len(groupIds) > 0 {
				for _, gid := range groupIds {
					_, _ = tx.Model("unsubscribe_records").InsertIgnore(g.Map{
						"email":            email,
						"group_id":         gid,
						"template_id":      0,
						"task_id":          0,
						"unsubscribe_time": now,
					})
					contact_activity.UpdateActivityByEmailAndGroup(email, gid)
				}
			} else {
				// Record unsubscription even if not currently in a group, ensuring future exclusion
				_, _ = tx.Model("unsubscribe_records").InsertIgnore(g.Map{
					"email":            email,
					"group_id":         0,
					"template_id":      0,
					"task_id":          0,
					"unsubscribe_time": now,
				})
			}
		}
		return nil
	})

	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to process unsubscribe: {}", err.Error())))
		return res, nil
	}

	res.Data.UnsubscribedEmails = emails
	res.Data.AffectedContacts = totalAffected
	res.Data.Count = len(emails)
	res.SetSuccess(public.LangCtx(ctx, "Successfully unsubscribed %d email(s)", len(emails)))

	_ = public.WriteLog(ctx, public.LogParams{
		Type: consts.LOGTYPE.Contacts,
		Log:  fmt.Sprintf("API Unsubscribed %d email(s): %s", len(emails), strings.Join(emails, ", ")),
		Data: g.Map{
			"emails":           emails,
			"affected_records": totalAffected,
			"reason":           req.Reason,
		},
	})

	return res, nil
}
