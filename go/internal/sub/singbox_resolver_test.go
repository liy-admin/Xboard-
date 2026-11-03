package sub

import (
	"testing"

	"xboard/internal/model"
)

func TestSingBoxDeclaresDefaultDomainResolver(t *testing.T) {
	// 加密订阅端点：flag=sing-box、无版本号（hiddify-core 1.13）
	ctx := &Ctx{User: &model.User{}, Servers: fixture(), ClientName: "sing-box", UserAgent: "sing-box", AppName: "X"}
	cfg, _, err := BuildSingBox(ctx)
	if err != nil {
		t.Fatal(err)
	}
	route, _ := cfg.(Map)["route"].(Map)
	if route["default_domain_resolver"] != "local" {
		t.Fatalf("route.default_domain_resolver = %v, want the direct dns tag \"local\"", route["default_domain_resolver"])
	}
	// 该标签必须真实存在于 dns.servers 且不经代理
	found := false
	for _, s := range Get(cfg, "dns.servers").([]any) {
		m := s.(Map)
		if m["tag"] == "local" && m["detour"] == "direct" {
			found = true
		}
	}
	if !found {
		t.Fatal("resolver tag must reference an existing direct dns server")
	}
}

func TestSingBoxOldCoreSkipsDefaultDomainResolver(t *testing.T) {
	ctx := &Ctx{User: &model.User{}, Servers: fixture(), ClientName: "sing-box", ClientVersion: "1.11.0", UserAgent: "sing-box 1.11.0", AppName: "X"}
	cfg, _, _ := BuildSingBox(ctx)
	route, _ := cfg.(Map)["route"].(Map)
	if _, ok := route["default_domain_resolver"]; ok {
		t.Fatal("sing-box < 1.12 does not know default_domain_resolver; must not be emitted")
	}
}
