package shop

import (
	"encoding/json"
	"testing"
	"time"

	"xboard/internal/model"
)

func TestPeriodKey(t *testing.T) {
	cases := map[string]string{"month_price": PeriodMonthly, "monthly": PeriodMonthly, "onetime_price": PeriodOnetime}
	for in, want := range cases {
		if got := PeriodKey(in); got != want {
			t.Errorf("PeriodKey(%q)=%q want %q", in, got, want)
		}
	}
}

func TestGbToTransfer(t *testing.T) {
	if GbToTransfer(1) != 1073741824 || GbToTransfer(0) != 0 {
		t.Error("gb conversion")
	}
	if GbToTransfer(1<<40) != 1<<63-1 {
		t.Error("overflow must cap at MaxInt64")
	}
}

func TestNextMonthlyOverflowMatchesPHP(t *testing.T) {
	// 与 PHP Carbon 的 day(31) 行为一致：2 月 5 日时当月目标日溢出为 3 月 3 日（原注释所述的月末截断并未生效）。
	expired := time.Date(2026, 1, 31, 10, 0, 0, 0, Loc)
	now := time.Date(2026, 2, 5, 0, 0, 0, 0, Loc)
	got := nextMonthly(expired, now)
	if got.Month() != time.March || got.Day() != 3 {
		t.Errorf("got %v", got)
	}
}

func TestNextMonthlyFutureThisMonth(t *testing.T) {
	expired := time.Date(2026, 1, 20, 0, 0, 0, 0, Loc)
	now := time.Date(2026, 3, 5, 0, 0, 0, 0, Loc)
	if got := nextMonthly(expired, now); got.Day() != 20 || got.Month() != time.March {
		t.Errorf("got %v", got)
	}
}

func TestPlanPricesDropsNullSoUnpricedPeriodIsNotFree(t *testing.T) {
	// 回归：prices 里显式 null 表示该周期未定价。曾被解码成 0 并判为可购买 = 0 元下单。
	p := &model.Plan{Prices: json.RawMessage(`{"monthly":null,"yearly":99.5,"onetime":0}`)}
	prices := PlanPrices(p)
	if _, ok := prices["monthly"]; ok {
		t.Fatal("null price must be treated as not purchasable")
	}
	if prices["yearly"] != 99.5 {
		t.Fatalf("yearly = %v", prices["yearly"])
	}
	// 明确定价为 0 的周期（免费套餐）仍然存在，与 null 区分开。
	if v, ok := prices["onetime"]; !ok || v != 0 {
		t.Fatalf("explicit 0 must be kept: %v %v", v, ok)
	}

	s := &Service{}
	u := &model.User{}
	if err := s.ValidatePurchase(u, &model.Plan{ID: 1, Show: true, Renew: true, Prices: p.Prices}, "monthly"); err == nil {
		t.Fatal("buying a null-priced period must be rejected")
	}
	if err := s.ValidatePurchase(u, &model.Plan{ID: 1, Show: true, Renew: true, Prices: p.Prices}, "month_price"); err == nil {
		t.Fatal("legacy period name must be rejected too")
	}
}

func planWithPrices(t *testing.T, prices string) *model.Plan {
	t.Helper()
	return &model.Plan{Prices: []byte(prices)}
}

func TestPlanDailyPrice(t *testing.T) {
	p := planWithPrices(t, `{"weekly":14,"quarterly":77}`)
	if got := PlanDailyPrice(p, PeriodWeekly); got != 2 {
		t.Errorf("weekly daily = %v, want 2", got)
	}
	// 指定周期未定价时，取各周期中最低的日单价（季付 77/90 < 周付 2）
	if got := PlanDailyPrice(p, PeriodYearly); got != 77.0/90 {
		t.Errorf("fallback daily = %v, want %v", got, 77.0/90)
	}
	if got := PlanDailyPrice(planWithPrices(t, `{}`), PeriodWeekly); got != 0 {
		t.Errorf("unpriced plan = %v, want 0", got)
	}
}

func TestConvertRemainingKeepsValue(t *testing.T) {
	bronze := planWithPrices(t, `{"weekly":14}`) // 2¥/天
	plat := planWithPrices(t, `{"weekly":21}`)   // 3¥/天
	day := int64(86400)
	// 14 天青铜（28¥）→ 铂金 9.33 天
	if got := ConvertRemaining(14*day, bronze, plat, PeriodWeekly); got != int64(float64(14*day)*2/3) {
		t.Errorf("bronze->plat = %v", got)
	}
	// 反向：3 天铂金（9¥）→ 青铜 4.5 天
	if got := ConvertRemaining(3*day, plat, bronze, PeriodWeekly); got != int64(float64(3*day)*1.5) {
		t.Errorf("plat->bronze = %v", got)
	}
	// 已过期 / 无剩余
	if got := ConvertRemaining(-5, bronze, plat, PeriodWeekly); got != 0 {
		t.Errorf("expired = %v", got)
	}
	// 旧套餐已删除：原样保留
	if got := ConvertRemaining(day, nil, plat, PeriodWeekly); got != day {
		t.Errorf("deleted old plan = %v", got)
	}
	// 旧套餐无定价（免费试用）：没有价值，不折算
	if got := ConvertRemaining(day, planWithPrices(t, `{}`), plat, PeriodWeekly); got != 0 {
		t.Errorf("free trial = %v", got)
	}
}
