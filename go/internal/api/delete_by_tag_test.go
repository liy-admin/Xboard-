package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"xboard/internal/setting"
)

func TestTagConditionQuotesExactly(t *testing.T) {
	cases := map[string]string{
		"Balala": `"Balala"`,
		`a"b`:    `"a\"b"`,
		"机场A":    `"机场A"`,
	}
	for in, want := range cases {
		if got := tagCondition(in); got != want {
			t.Errorf("tagCondition(%q)=%s want %s", in, got, want)
		}
	}
}

func TestDeleteByTagRequiresTag(t *testing.T) {
	// 没有订阅标识时必须拒绝，绝不能退化成「删除全部节点」
	gin.SetMode(gin.TestMode)
	a := &App{Set: setting.New(nil)}
	r := gin.New()
	r.POST("/d", a.nodeDeleteByTag)
	r.GET("/c", a.nodeCountByTag)
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/d", strings.NewReader(`{"tag":"   "}`)),
		httptest.NewRequest(http.MethodPost, "/d", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodGet, "/c?tag=", nil),
	} {
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Errorf("%s %s: status %d body %s", req.Method, req.URL, w.Code, w.Body.String())
		}
	}
}
