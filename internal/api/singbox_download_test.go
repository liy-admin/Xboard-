package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"xboard/internal/setting"
)

func singboxRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	a := &App{Set: setting.New(nil)}
	r := gin.New()
	r.POST("/d", a.singboxDownload)
	r.GET("/s", a.singboxDownloadStatus)
	return r
}

func TestSingboxDownloadRejectsBadVersion(t *testing.T) {
	// 版本号会拼进下载地址与磁盘路径，非法值必须在发起任何下载前被拒绝。
	for _, v := range []string{"../etc", "1.2", "x; rm -rf /", "1.0.0/../../x"} {
		req := httptest.NewRequest(http.MethodPost, "/d", strings.NewReader(`{"version":"`+v+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		singboxRouter().ServeHTTP(w, req)
		if w.Code != 422 {
			t.Errorf("version %q: status %d body %s", v, w.Code, w.Body.String())
		}
	}
}

func TestSingboxDownloadStatusShape(t *testing.T) {
	w := httptest.NewRecorder()
	singboxRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/s", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"state"`) {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}
