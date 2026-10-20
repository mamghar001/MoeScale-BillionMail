package settings

import (
	"billionmail-core/api/settings/v1"
	"billionmail-core/internal/service/public"
	"context"
)

func (c *ControllerV1) SetCircuitBreaker(ctx context.Context, req *v1.SetCircuitBreakerReq) (res *v1.SetCircuitBreakerRes, err error) {
	res = &v1.SetCircuitBreakerRes{}

	if req.Enabled != nil {
		if err := public.OptionsMgrInstance.SetOption(ctx, "circuit_breaker_enabled", *req.Enabled); err != nil {
			res.SetError(err)
			return res, nil
		}
	}

	if req.Threshold > 0 {
		th := req.Threshold
		if th > 1.0 {
			th = th / 100.0
		}
		if err := public.OptionsMgrInstance.SetOption(ctx, "circuit_breaker_threshold", th); err != nil {
			res.SetError(err)
			return res, nil
		}
	}

	res.SetSuccess(public.LangCtx(ctx, "Circuit breaker settings updated successfully"))
	return res, nil
}
