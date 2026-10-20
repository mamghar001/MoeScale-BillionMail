package settings

import (
	"billionmail-core/api/settings/v1"
	"billionmail-core/internal/service/public"
	"context"
)

func (c *ControllerV1) GetCircuitBreaker(ctx context.Context, req *v1.GetCircuitBreakerReq) (res *v1.GetCircuitBreakerRes, err error) {
	res = &v1.GetCircuitBreakerRes{}

	var enabled bool = true
	_ = public.OptionsMgrInstance.GetOption(ctx, "circuit_breaker_enabled", &enabled)

	var threshold float64 = 0.15
	_ = public.OptionsMgrInstance.GetOption(ctx, "circuit_breaker_threshold", &threshold)
	if threshold <= 0 {
		threshold = 0.15
	}

	res.Data.Enabled = enabled
	res.Data.Threshold = threshold

	res.SetSuccess(public.LangCtx(ctx, "Circuit breaker settings retrieved successfully"))
	return res, nil
}
