package sub

import (
	"strings"
	"testing"

	"xboard/internal/model"
)

func TestInfoNodeShowsUnlimitedTraffic(t *testing.T) {
	servers := []Map{{"id": 1, "type": "trojan", "name": "HK-1", "host": "h", "port": 443}}
	unlimited := &Ctx{User: &model.User{TransferEnab: 0, U: 5, D: 7}}
	got := AddInfoServers(servers, unlimited, 0, true)
	if !strings.Contains(Str(got[0]["name"]), "剩余流量：无限") {
		t.Fatalf("unlimited user must see 无限, got %q", got[0]["name"])
	}

	limited := &Ctx{User: &model.User{TransferEnab: 10 << 30, U: 1 << 30}}
	got = AddInfoServers(servers, limited, 0, true)
	if !strings.Contains(Str(got[0]["name"]), "剩余流量：9") {
		t.Fatalf("limited user keeps the remaining amount, got %q", got[0]["name"])
	}
}
