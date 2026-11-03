package jobs

import (
	"strings"
	"testing"

	"xboard/internal/model"
)

func TestFindProbeUserWhyImportedNodeWithoutGroup(t *testing.T) {
	// 没有分组的导入节点：认证信息来自 code，不需要面板用户，不应被判为「无法测试」。
	code := "0f0b0c0d-1111-2222-3333-444444444444"
	e := &Env{} // 无分组路径不访问数据库
	u, ok, why := e.findProbeUserWhy(model.Server{Code: &code})
	if !ok || u == nil || why != "" {
		t.Fatalf("imported node must be probeable: ok=%v why=%q", ok, why)
	}
}

func TestFindProbeUserWhyNativeNodeWithoutGroupExplains(t *testing.T) {
	e := &Env{}
	_, ok, why := e.findProbeUserWhy(model.Server{})
	if ok || !strings.Contains(why, "权限组") {
		t.Fatalf("native node without group must explain why: ok=%v why=%q", ok, why)
	}
	empty := "  "
	if _, ok, _ := e.findProbeUserWhy(model.Server{Code: &empty}); ok {
		t.Fatal("blank code is not a credential")
	}
}
