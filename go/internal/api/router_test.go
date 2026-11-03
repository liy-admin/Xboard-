package api

import (
	"testing"

	"xboard/internal/setting"
	"xboard/internal/store"
)

// 路由注册阶段不访问数据库；这里确保所有路由组合在一起时不会因重复注册而 panic。
func TestRouterBuildsWithoutDuplicateRoutes(t *testing.T) {
	a := &App{S: &store.Store{}, Set: setting.New(nil), AppKey: "k"}
	if a.Router() == nil {
		t.Fatal("router is nil")
	}
	// 后台引擎按需构建，这里显式构建以检查管理端路由是否重复注册。
	if a.buildAdminEngine() == nil {
		t.Fatal("admin engine is nil")
	}
}
