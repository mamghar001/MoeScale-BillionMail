package batch_mail

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"billionmail-core/api/batch_mail/v1"
)

func (c *ControllerV1) GetBusinessHours(ctx context.Context, req *v1.GetBusinessHoursReq) (res *v1.GetBusinessHoursRes, err error) {
	res = &v1.GetBusinessHoursRes{}

	enabled := 0
	val, err := g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_enabled").Value("value")
	if err == nil && !val.IsEmpty() && val.String() == "1" {
		enabled = 1
	}

	start := "08:00"
	val, err = g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_start").Value("value")
	if err == nil && !val.IsEmpty() {
		start = val.String()
	}

	end := "18:00"
	val, err = g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_end").Value("value")
	if err == nil && !val.IsEmpty() {
		end = val.String()
	}

	tz := "America/New_York"
	val, err = g.DB().Model("bm_options").Ctx(ctx).Where("name", "business_hours_tz").Value("value")
	if err == nil && !val.IsEmpty() {
		tz = val.String()
	}

	res.Data.Enabled = enabled
	res.Data.Start = start
	res.Data.End = end
	res.Data.Timezone = tz
	res.SetSuccess("success")
	return res, nil
}

func (c *ControllerV1) SetBusinessHours(ctx context.Context, req *v1.SetBusinessHoursReq) (res *v1.SetBusinessHoursRes, err error) {
	res = &v1.SetBusinessHoursRes{}

	tz := req.Timezone
	if tz == "" {
		tz = "America/New_York"
	}

	enabledStr := "0"
	if req.Enabled == 1 {
		enabledStr = "1"
	}

	items := map[string]string{
		"business_hours_enabled": enabledStr,
		"business_hours_start":   req.Start,
		"business_hours_end":     req.End,
		"business_hours_tz":      tz,
	}

	for k, v := range items {
		count, _ := g.DB().Model("bm_options").Ctx(ctx).Where("name", k).Count()
		if count > 0 {
			_, err = g.DB().Model("bm_options").Ctx(ctx).Where("name", k).Data(g.Map{"value": v}).Update()
		} else {
			_, err = g.DB().Model("bm_options").Ctx(ctx).Data(g.Map{"name": k, "value": v}).Insert()
		}
		if err != nil {
			res.SetError(err)
			return res, err
		}
	}

	res.SetSuccess("Business hours updated successfully")
	return res, nil
}
