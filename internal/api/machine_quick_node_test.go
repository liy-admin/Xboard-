package api

import (
	"bytes"
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRealityKeyPairIsValidX25519(t *testing.T) {
	for i := 0; i < 20; i++ {
		priv, pub, err := realityKeyPair()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := base64.RawURLEncoding.DecodeString(priv)
		if err != nil || len(raw) != 32 {
			t.Fatalf("private key must be 32 bytes raw-url base64: %q (%v)", priv, err)
		}
		if raw[0]&7 != 0 || raw[31]&128 != 0 || raw[31]&64 == 0 {
			t.Fatalf("private key is not clamped: %x", raw)
		}
		key, err := ecdh.X25519().NewPrivateKey(raw)
		if err != nil {
			t.Fatal(err)
		}
		if want := base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()); pub != want {
			t.Fatalf("public key does not match private key: got %s want %s", pub, want)
		}
	}
}

func TestQuickNodeGroupIDs(t *testing.T) {
	var m map[string]any
	dec := json.NewDecoder(bytes.NewReader([]byte(`{"g": [1, "2", 2, 0, -3, "x", 5.0]}`)))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	got := quickNodeGroupIDs(m["g"])
	want := []int64{1, 2, 5}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if len(quickNodeGroupIDs(nil)) != 0 || len(quickNodeGroupIDs("1")) != 0 {
		t.Fatal("non-array input must yield no groups")
	}
}

func TestValidNodeHost(t *testing.T) {
	good := []string{"203.0.113.7", "2001:db8::1", "node.example.com", "a-b.example.co", "www.microsoft.com"}
	bad := []string{"", "localhost", "not a host", "-a.example.com", "a..example.com", "exa_mple.com", "a.example.com/x"}
	for _, h := range good {
		if !validNodeHost(h) {
			t.Errorf("%q should be valid", h)
		}
	}
	for _, h := range bad {
		if validNodeHost(h) {
			t.Errorf("%q should be invalid", h)
		}
	}
}

func TestMachineQueryIDAcceptsIDAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for url, want := range map[string]int64{
		"/x?machine_id=7":      7,
		"/x?id=9":              9,
		"/x?machine_id=3&id=9": 3,
		"/x":                   0,
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", url, nil)
		if got := machineQueryID(c); got != want {
			t.Errorf("%s: got %d want %d", url, got, want)
		}
	}
}
