package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// capturePushes 把实际发送替换成记录，返回读取记录的函数。
func capturePushes(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var got []string
	oldDelay, oldHook := clientPushDelay, pushNowHook
	clientPushDelay = 60 * time.Millisecond
	pushNowHook = func(_ *App, reason string) {
		mu.Lock()
		got = append(got, reason)
		mu.Unlock()
	}
	t.Cleanup(func() { clientPushDelay, pushNowHook = oldDelay, oldHook })
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
}

func TestNotifyClientsSoonCoalesces(t *testing.T) {
	read := capturePushes(t)
	a := &App{}
	// 连续保存多个节点 + 改一次套餐：窗口内只推一次，原因去重排序
	a.notifyClientsSoon("node")
	a.notifyClientsSoon("node")
	a.notifyClientsSoon("plan")
	a.notifyClientsSoon("node")
	time.Sleep(250 * time.Millisecond)
	if got := read(); len(got) != 1 || got[0] != "node,plan" {
		t.Fatalf("want a single push 'node,plan', got %v", got)
	}
	// 窗口过后的新改动会再推一次
	a.notifyClientsSoon("import")
	time.Sleep(250 * time.Millisecond)
	if got := read(); len(got) != 2 || got[1] != "import" {
		t.Fatalf("second burst: %v", got)
	}
}

func TestNotifyClientsSoonIsTrailing(t *testing.T) {
	read := capturePushes(t)
	a := &App{}
	// 每次改动都推迟发送：只要还在持续改，就不会提前推出中间状态
	for i := 0; i < 4; i++ {
		a.notifyClientsSoon("node")
		time.Sleep(30 * time.Millisecond) // 小于窗口
	}
	if got := read(); len(got) != 0 {
		t.Fatalf("must not push while edits keep coming, got %v", got)
	}
	time.Sleep(250 * time.Millisecond)
	if got := read(); len(got) != 1 {
		t.Fatalf("want exactly one push after edits stop, got %v", got)
	}
}

func TestPushAfterOnlyOnSuccess(t *testing.T) {
	read := capturePushes(t)
	gin.SetMode(gin.TestMode)
	a := &App{}
	r := gin.New()
	r.POST("/ok", a.pushAfter("node"), func(c *gin.Context) { c.JSON(200, gin.H{"data": true}) })
	r.POST("/bad", a.pushAfter("node"), func(c *gin.Context) { c.AbortWithStatusJSON(400, gin.H{"message": "x"}) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/bad", nil))
	time.Sleep(200 * time.Millisecond)
	if got := read(); len(got) != 0 {
		t.Fatalf("failed request must not push, got %v", got)
	}
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/ok", nil))
	time.Sleep(200 * time.Millisecond)
	if got := read(); len(got) != 1 || got[0] != "node" {
		t.Fatalf("successful request must push once, got %v", got)
	}
}
