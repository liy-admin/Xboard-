package api

import "testing"

// 回归：连接关闭后再投递（例如鉴权超时协程）不得 panic。
func TestEnqueueAfterCloseDoesNotPanic(t *testing.T) {
	c := &wsConn{send: make(chan []byte, 4)}
	if !c.enqueue([]byte("a")) {
		t.Fatal("enqueue before close must succeed")
	}
	c.close()
	if c.enqueue([]byte("b")) {
		t.Fatal("enqueue after close must report false")
	}
	c.close() // 重复关闭安全
}
