package api

import "testing"

// 回归：ensureHub 必须真正创建注册表，否则鉴权成功的连接会在 addClient / addNode 处 nil 解引用。
func TestEnsureHubCreatesRegistry(t *testing.T) {
	a := &App{}
	a.ensureHub()
	if a.hub == nil {
		t.Fatal("hub must be created")
	}
	first := a.hub
	a.ensureHub()
	if a.hub != first {
		t.Fatal("ensureHub must be idempotent")
	}
	c := &wsConn{send: make(chan []byte, 1)}
	a.hub.addClient(7, c)
	a.hub.addNode(9, c)
	if a.hub.node(9) != c {
		t.Fatal("node registry broken")
	}
}
