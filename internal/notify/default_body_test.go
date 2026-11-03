package notify

import (
	"strings"
	"testing"
)

func TestDefaultMailBodyCarriesKeyContent(t *testing.T) {
	v := map[string]any{"name": "App", "url": "https://x.test", "code": "123456", "link": "https://x.test/l?t=1", "content": "Hello"}
	cases := map[string]string{
		"verify":        "123456",
		"mailLogin":     "https://x.test/l?t=1",
		"notify":        "Hello",
		"remindExpire":  "24 小时",
		"remindTraffic": "流量",
	}
	for tpl, want := range cases {
		if body := DefaultMailBody(tpl, v); !strings.Contains(body, want) {
			t.Errorf("%s fallback body lacks %q: %s", tpl, want, body)
		}
	}
}

func TestDefaultMailBodyEscapesValues(t *testing.T) {
	body := DefaultMailBody("verify", map[string]any{"name": "<b>x</b>", "code": "<script>1</script>"})
	if strings.Contains(body, "<script>") {
		t.Fatalf("unescaped value in body: %s", body)
	}
}
