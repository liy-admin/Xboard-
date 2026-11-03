package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"xboard/internal/setting"
)

func bindRegister(t *testing.T, body string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var msg string
	r := gin.New()
	r.POST("/r", func(c *gin.Context) {
		var in struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
		}
		if err := c.ShouldBind(&in); err != nil {
			msg = registerBindMessage(err)
		}
	})
	req := httptest.NewRequest(http.MethodPost, "/r", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)
	return msg
}

func TestRegisterBindMessage(t *testing.T) {
	cases := map[string]string{
		`{"email":"a@b.co","password":"short"}`:          "密码长度不能少于 8 位",
		`{"email":"not-an-email","password":"12345678"}`: "邮箱格式不正确",
		`{"password":"12345678"}`:                        "邮箱不能为空",
		`{"email":"a@b.co"}`:                             "密码不能为空",
		`not json`:                                       "请求格式错误，请检查参数（email、password）",
	}
	for body, want := range cases {
		if got := bindRegister(t, body); got != want {
			t.Errorf("%s => %q, want %q", body, got, want)
		}
	}
}

func TestSubURLFallsBackToRequestHost(t *testing.T) {
	a := &App{Set: setting.New(nil)}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(a.rememberBase)
	r.GET("/x", func(c *gin.Context) { c.String(200, a.encSubscribeURL("tok")) })
	req := httptest.NewRequest(http.MethodGet, "http://192.168.31.239:7001/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Body.String(); got != "http://192.168.31.239:7001/s-enc/tok" {
		t.Errorf("got %q", got)
	}
}

func TestRegisterWebBlocked(t *testing.T) {
	open := &App{Set: setting.NewFromMap(map[string]string{"user_web_login_enable": "1"})}
	closed := &App{Set: setting.NewFromMap(map[string]string{"user_web_login_enable": "0"})}
	cases := []struct {
		a        *App
		deviceID string
		want     bool
	}{
		{open, "", false},
		{open, "dev-1", false},
		{closed, "", true},
		{closed, "   ", true},
		{closed, "dev-1", false}, // App 端带 device_id 注册不应被拦
	}
	for i, tc := range cases {
		if got := tc.a.registerWebBlocked(tc.deviceID); got != tc.want {
			t.Errorf("case %d (device %q): got %v want %v", i, tc.deviceID, got, tc.want)
		}
	}
}

func TestMarkTrialClaimedUpsertsClaimedFlag(t *testing.T) {
	_, db := dryRunApp(t)
	stmt := markTrialClaimed(db, " dev-1 ", 42).Statement
	sql := stmt.SQL.String()
	for _, want := range []string{
		"INSERT INTO v2_device_binding",
		"claimed_trial",
		"ON DUPLICATE KEY UPDATE claimed_trial = 1",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("sql missing %q: %s", want, sql)
		}
	}
	if len(stmt.Vars) != 6 || stmt.Vars[0] != "dev-1" || stmt.Vars[1] != int64(42) || stmt.Vars[4] != int64(42) {
		t.Errorf("unexpected vars: %v", stmt.Vars)
	}
}
