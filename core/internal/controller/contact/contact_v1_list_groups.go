package contact

import (
	"billionmail-core/api/contact/v1"
	"billionmail-core/internal/service/contact"
	"billionmail-core/internal/service/public"
	"context"
	"fmt"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) ListGroups(ctx context.Context, req *v1.ListGroupsReq) (res *v1.ListGroupsRes, err error) {

	res = &v1.ListGroupsRes{}
	// Handle request parameters
	page := req.Page
	if page <= 0 {
		page = DefaultPage
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	// Get group list
	groups, total, err := contact.ContactsGroupWithPage(ctx, page, pageSize, req.Keyword)
	if err != nil {
		res.Code = 500
		res.SetError(gerror.New(public.LangCtx(ctx, "Failed to get group list {}", err.Error())))
		return
	}
	//hostUrl := domains.GetBaseURL()
	// Convert group information
	groupInfos := make([]*v1.ContactGroupInfo, 0, len(groups))

	// Batch count active and unsubscribed contacts for all groups on the page in a single query
	groupIds := make([]int, len(groups))
	for i, group := range groups {
		groupIds[i] = group.Id
	}
	type GroupCount struct {
		GroupId int `json:"group_id"`
		Active  int `json:"active"`
		Count   int `json:"count"`
	}
	var counts []GroupCount
	if len(groupIds) > 0 {
		_ = g.DB().Model("bm_contacts").Ctx(ctx).
			Fields("group_id, active, count(*) as count").
			WhereIn("group_id", groupIds).
			Group("group_id, active").
			Scan(&counts)
	}
	countMap := make(map[string]int)
	for _, c := range counts {
		countMap[fmt.Sprintf("%d_%d", c.GroupId, c.Active)] = c.Count
	}

	for _, group := range groups {
		activeCount := countMap[fmt.Sprintf("%d_1", group.Id)]
		unsubscribeCount := countMap[fmt.Sprintf("%d_0", group.Id)]

		// If the group token is empty, generate a new token
		if group.Token == "" {

			token := GfMd5Short()
			group.Token = token
			_, err := g.DB().Model("bm_contact_groups").
				Data(g.Map{"token": token}).
				Where("id", group.Id).
				Update()
			if err != nil {
				g.Log().Error(ctx, "Failed to update the group token: %v", err)
			}
		}
		//if group.WelcomeHtml == "" {
		//	html, drag := subscribe_list.GetDefaultTemplate(1)
		//	group.WelcomeHtml = html
		//	group.WelcomeDrag = drag
		//}
		//if group.ConfirmHtml == "" {
		//	html, drag := subscribe_list.GetDefaultTemplate(2)
		//	group.ConfirmHtml = html
		//	group.ConfirmDrag = drag
		//}
		//if group.SuccessUrl == "" {
		//	group.SuccessUrl = hostUrl + "/subscribe_success.html"
		//}
		//if group.ConfirmUrl == "" {
		//	group.ConfirmUrl = hostUrl + "/subscribe_confirm.html"
		//}
		//if group.AlreadyUrl == "" {
		//	group.AlreadyUrl = hostUrl + "/already_subscribed.html"
		//}
		//if group.SubscribeForm == "" {
		//	group.SubscribeForm = subscribe_list.GetSubscribeFormCode(group.Token)
		//}
		//if group.WelcomeSubject == "" {
		//	group.WelcomeSubject = "Welcome Aboard!"
		//}
		//if group.ConfirmSubject == "" {
		//	group.ConfirmSubject = "Confirm Your Subscription"
		//}

		contactGroup := v1.ContactGroup{
			Id:               group.Id,
			Name:             group.Name,
			Description:      group.Description,
			CreateTime:       group.CreateTime,
			UpdateTime:       group.UpdateTime,
			Token:            group.Token,
			DoubleOptin:      group.DoubleOptin,
			WelcomeHtml:      group.WelcomeHtml,
			WelcomeDrag:      group.WelcomeDrag,
			ConfirmHtml:      group.ConfirmHtml,
			ConfirmDrag:      group.ConfirmDrag,
			SuccessUrl:       group.SuccessUrl,
			ConfirmUrl:       group.ConfirmUrl,
			AlreadyUrl:       group.AlreadyUrl,
			SubscribeForm:    group.SubscribeForm,
			ConfirmSubject:   group.ConfirmSubject,
			WelcomeSubject:   group.WelcomeSubject,
			SendWelcomeEmail: group.SendWelcomeEmail,
		}

		// Convert to group information
		groupInfo := &v1.ContactGroupInfo{
			ContactGroup:     contactGroup,
			TotalCount:       activeCount + unsubscribeCount,
			ActiveCount:      activeCount,
			UnsubscribeCount: unsubscribeCount,
			//SubscribeLink:    hostUrl + "/subscribe_form.html?token=" + group.Token,
		}
		groupInfos = append(groupInfos, groupInfo)
	}

	res.Data.Total = total
	res.Data.List = groupInfos
	res.SetSuccess(public.LangCtx(ctx, "Group list retrieved successfully"))
	return

}
