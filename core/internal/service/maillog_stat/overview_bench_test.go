package maillog_stat

import (
	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"billionmail-core/internal/service/database_initialization"
	"testing"
	"time"
)

func TestOverviewBench(t *testing.T) {
	if err := database_initialization.InitDatabase(); err != nil {
		t.Fatal(err)
	}

	o := NewOverview()
	now := time.Now().Unix()
	t0 := time.Now()
	res := o.Overview(0, "scalewarm.shop", now-86400, now)
	t.Logf("Finished in %v: %+v", time.Since(t0), res["dashboard"])
}
