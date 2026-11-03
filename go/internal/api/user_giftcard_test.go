package api

import (
	"encoding/json"
	"testing"

	"xboard/internal/model"
)

func tplOf(t *testing.T, typ int, rewards, conds, limits, special string) *model.GiftCardTemplate {
	t.Helper()
	tp := &model.GiftCardTemplate{Type: typ, Status: true}
	if rewards != "" {
		tp.Rewards = json.RawMessage(rewards)
	}
	if conds != "" {
		tp.Conditions = json.RawMessage(conds)
	}
	if limits != "" {
		tp.Limits = json.RawMessage(limits)
	}
	if special != "" {
		tp.SpecialConfig = json.RawMessage(special)
	}
	return tp
}

func TestGiftCodeAvailable(t *testing.T) {
	now := int64(1000)
	exp := int64(999)
	c := &model.GiftCardCode{Status: giftCodeUnused, MaxUsage: 1}
	if !giftCodeAvailable(c, now) {
		t.Fatal("fresh code should be available")
	}
	c.ExpiresAt = &exp
	if giftCodeAvailable(c, now) {
		t.Fatal("expired code must be unavailable")
	}
	c2 := &model.GiftCardCode{Status: giftCodeUsed, UsageCount: 1, MaxUsage: 1}
	if giftCodeAvailable(c2, now) {
		t.Fatal("used-up code must be unavailable")
	}
}

func TestGeneralCardNeedsPlanForTransferRewards(t *testing.T) {
	tp := tplOf(t, giftTypeGeneral, `{"transfer_enable": 10}`, "", "", "")
	noPlan := &model.User{}
	if templateUserConditions(tp, noPlan, 1000, false) {
		t.Fatal("user without plan must not redeem transfer reward")
	}
	pid := int64(3)
	withPlan := &model.User{PlanID: &pid}
	if !templateUserConditions(tp, withPlan, 1000, false) {
		t.Fatal("user with plan should be eligible")
	}
}

func TestPlanCardRejectsActiveUser(t *testing.T) {
	tp := tplOf(t, giftTypePlan, `{"plan_id": 2}`, "", "", "")
	pid := int64(1)
	exp := int64(9999)
	active := &model.User{PlanID: &pid, ExpiredAt: &exp}
	if templateUserConditions(tp, active, 1000, false) {
		t.Fatal("active subscriber should not get plan card")
	}
	if !templateUserConditions(tp, &model.User{}, 1000, false) {
		t.Fatal("non-member should be eligible")
	}
}

func TestConditionsNewUserPaidAllowedInvite(t *testing.T) {
	now := int64(10 * 86400)
	u := &model.User{CreatedAt: now - 2*86400}
	if !templateUserConditions(tplOf(t, giftTypeGeneral, `{}`, `{"new_user_only": true}`, "", ""), u, now, false) {
		t.Fatal("new user within 7 days should pass")
	}
	old := &model.User{CreatedAt: now - 30*86400}
	if templateUserConditions(tplOf(t, giftTypeGeneral, `{}`, `{"new_user_only": true}`, "", ""), old, now, false) {
		t.Fatal("old user should fail new_user_only")
	}
	if templateUserConditions(tplOf(t, giftTypeGeneral, `{}`, `{"paid_user_only": true}`, "", ""), u, now, false) {
		t.Fatal("paid_user_only without paid order must fail")
	}
	if templateUserConditions(tplOf(t, giftTypeGeneral, `{}`, `{"require_invite": true}`, "", ""), u, now, false) {
		t.Fatal("require_invite without inviter must fail")
	}
	pid := int64(7)
	if templateUserConditions(tplOf(t, giftTypeGeneral, `{}`, `{"allowed_plans": [1,2]}`, "", ""), &model.User{PlanID: &pid}, now, false) {
		t.Fatal("plan 7 not in allowed list")
	}
	if !templateUserConditions(tplOf(t, giftTypeGeneral, `{}`, `{"allowed_plans": [1,7]}`, "", ""), &model.User{PlanID: &pid}, now, false) {
		t.Fatal("plan 7 in allowed list should pass")
	}
}

func TestUsageLimitAndCooldown(t *testing.T) {
	tp := tplOf(t, giftTypeGeneral, `{}`, "", `{"max_use_per_user": 2, "cooldown_hours": 1}`, "")
	if templateUsageAllowed(tp, 2, 0, 1000) {
		t.Fatal("max_use_per_user reached")
	}
	if templateUsageAllowed(tp, 1, 1000, 1000+1800) {
		t.Fatal("cooldown should block within 1h")
	}
	if !templateUsageAllowed(tp, 1, 1000, 1000+3601) {
		t.Fatal("cooldown should release after 1h")
	}
}

func TestMysteryPicksByWeight(t *testing.T) {
	tp := tplOf(t, giftTypeMystery,
		`{"random_rewards":[{"weight":1,"balance":100},{"weight":3,"balance":500}]}`, "", "", "")
	// pick 返回 [0,n)，+1 后落在 [1,4]：0 -> r=1 命中第一项，2 -> r=3 命中第二项。
	first := calcActualRewards(tp, 0, func(n int) int { return 0 })
	if first["balance"].(float64) != 100 {
		t.Fatalf("r=1 should hit weight-1 reward, got %v", first["balance"])
	}
	second := calcActualRewards(tp, 0, func(n int) int { return 2 })
	if second["balance"].(float64) != 500 {
		t.Fatalf("r=3 should hit weight-3 reward, got %v", second["balance"])
	}
	if _, ok := second["weight"]; ok {
		t.Fatal("weight key must be removed from result")
	}
}

func TestFestivalBonusSkipsPlanID(t *testing.T) {
	tp := tplOf(t, giftTypeGeneral,
		`{"balance": 100, "plan_id": 5, "invite_reward_rate": 0.5}`, "", "",
		`{"festival_bonus": 2, "start_time": 0, "end_time": 9999999999}`)
	got := calcActualRewards(tp, 1000, nil)
	if got["balance"].(int64) != 200 {
		t.Fatalf("balance should double, got %v", got["balance"])
	}
	if got["plan_id"].(float64) != 5 {
		t.Fatalf("plan_id must not be scaled, got %v", got["plan_id"])
	}
	if m := festivalMultiplier(tp, 1000); m != 2 {
		t.Fatalf("multiplier want 2 got %v", m)
	}
}

func TestFestivalOutsideWindowNoScaling(t *testing.T) {
	tp := tplOf(t, giftTypeGeneral, `{"balance": 100}`, "", "",
		`{"festival_bonus": 2, "start_time": 5000, "end_time": 6000}`)
	got := calcActualRewards(tp, 1000, nil)
	if got["balance"].(float64) != 100 {
		t.Fatalf("outside window must not scale, got %v", got["balance"])
	}
	if m := festivalMultiplier(tp, 1000); m != 1 {
		t.Fatalf("outside window multiplier should be 1, got %v", m)
	}
}

func TestUserIsActive(t *testing.T) {
	pid := int64(1)
	exp := int64(500)
	if userIsActive(&model.User{PlanID: &pid, ExpiredAt: &exp}, 1000) {
		t.Fatal("expired plan is not active")
	}
	if !userIsActive(&model.User{PlanID: &pid}, 1000) {
		t.Fatal("permanent plan is active")
	}
	if userIsActive(&model.User{PlanID: &pid, Banned: true}, 1000) {
		t.Fatal("banned user is not active")
	}
}
