package sub

import (
	"encoding/json"
	"testing"

	"xboard/internal/model"
)

func TestProbeOutboundUsesNodeCodeNotUserUUID(t *testing.T) {
	code := "0f0b0c0d-1111-2222-3333-444444444444"
	ps, _ := json.Marshal(map[string]any{"tls": map[string]any{"server_name": "a.example.com"}})
	srv := model.Server{ID: 1, Type: "anytls", Name: "n", Host: "a.example.com", Port: "443", ServerPort: 443,
		Code: &code, ProtocolSettings: ps}
	placeholder := &model.User{UUID: "00000000-0000-0000-0000-000000000000"}
	ob := ProbeOutbound(srv, srv, placeholder)
	if ob == nil {
		t.Fatal("outbound must be generated for an imported node with a placeholder user")
	}
	if ob["password"] != code {
		t.Fatalf("credential must come from code, got %v", ob["password"])
	}
}
