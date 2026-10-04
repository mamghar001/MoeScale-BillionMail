package contact

import (
	"context"

	"billionmail-core/api/contact/v1"
)

func (c *ControllerV1) ListContacts(ctx context.Context, req *v1.ListContactsReq) (res *v1.ListContactsRes, err error) {
	res = &v1.ListContactsRes{}

	if req.Page < 1 {
		req.Page = DefaultPage
	}
	if req.PageSize < 1 || req.PageSize > MaxPageSize {
		req.PageSize = DefaultPageSize
	}
	if req.Status < 0 || req.Status > 1 {
		req.Status = 1 // Default to active
	}

	ndpReq := &v1.ListContactsNDPReq{
		Page:     req.Page,
		PageSize: req.PageSize,
		GroupId:  req.GroupId,
		Keyword:  req.Keyword,
		Active:   req.Status,
	}

	ndpRes, err := c.ListContactsNDP(ctx, ndpReq)
	if err != nil {
		return nil, err
	}

	// Ensure groups array is populated for backward compatibility
	for _, contact := range ndpRes.Data.List {
		if len(contact.Groups) == 0 && contact.GroupId > 0 {
			contact.Groups = []v1.GroupInfo{
				{
					Id:   contact.GroupId,
					Name: contact.GroupName,
				},
			}
		}
	}

	res.StandardRes = ndpRes.StandardRes
	res.Data.Total = ndpRes.Data.Total
	res.Data.List = ndpRes.Data.List
	return res, nil
}
