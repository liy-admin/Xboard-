package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
)

func TestPlanResourceLegacyPricesAndCapacity(t *testing.T) {
	cap0 := int64(0)
	content := "流量 {{transfer}} 重置 {{reset_method}}"
	tr := int64(100)
	p := &model.Plan{
		ID: 1, Name: "P", Prices: json.RawMessage(`{"monthly": 9.9}`),
		CapacityLimit: &cap0, Content: &content, TransferEnable: &tr,
	}
	out := planResource(p, 1)
	if out["capacity_limit"] != "已售罄" {
		t.Fatalf("capacity <= 0 should be Sold out, got %v", out["capacity_limit"])
	}
	if out["month_price"].(float64) != 9.9*100 {
		t.Fatalf("month_price should be multiplied by 100, got %v", out["month_price"])
	}
	if out["year_price"] != nil {
		t.Fatalf("missing price must be null, got %v", out["year_price"])
	}
	if !strings.Contains(out["content"].(string), "流量 100 重置 按月") {
		t.Fatalf("content placeholders not replaced: %v", out["content"])
	}
}

func TestResetMethodTextFollowsSystemSetting(t *testing.T) {
	if resetMethodText(nil, 3) != "每年1月1日" {
		t.Fatal("nil method should follow system setting")
	}
	m := 2
	if resetMethodText(&m, 3) != "不重置" {
		t.Fatal("explicit method should win")
	}
}

func TestOrderedGroupsKeepsFirstAppearanceOrder(t *testing.T) {
	g := &orderedGroups{vals: map[string][]gin.H{}}
	g.add("zeta", gin.H{"id": 1})
	g.add("alpha", gin.H{"id": 2})
	g.add("zeta", gin.H{"id": 3})
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"zeta":[{"id":1},{"id":3}],"alpha":[{"id":2}]}` {
		t.Fatalf("order wrong: %s", b)
	}
}

func TestSubscribeURLBuilders(t *testing.T) {
	a := &App{Set: nil}
	if got := a.buildSubURL("/s/tok", "edge.example"); got != "https://edge.example/s/tok" {
		t.Fatalf("domain form: %s", got)
	}
}
