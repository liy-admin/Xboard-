package notify

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramSendMessageEncodesParams(t *testing.T) {
	var gotPath, gotText, gotMode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotText = r.URL.Query().Get("text")
		gotMode = r.URL.Query().Get("parse_mode")
		io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
	}))
	defer srv.Close()
	c := NewTelegram("TOKEN")
	c.BaseURL = srv.URL
	if err := c.SendMessage(42, "hello_world", "markdown"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/botTOKEN/sendMessage" {
		t.Fatalf("path %s", gotPath)
	}
	if gotText != `hello\_world` || gotMode != "markdown" {
		t.Fatalf("text=%q mode=%q", gotText, gotMode)
	}
}

func TestTelegramAPIErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":false,"description":"chat not found"}`)
	}))
	defer srv.Close()
	c := NewTelegram("T")
	c.BaseURL = srv.URL
	err := c.SendMessage(1, "x", "")
	if err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Fatalf("want api error, got %v", err)
	}
}
