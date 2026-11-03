package api

import (
	"crypto/sha1"
	"fmt"
	"testing"
)

// PHP: sha1(json_encode([])) == sha1("[]")
func TestNodeRevisionEmptyMatchesPHP(t *testing.T) {
	want := fmt.Sprintf("%x", sha1.Sum([]byte("[]")))
	if got := nodeRevision(nil); got != want {
		t.Fatalf("want %s got %s", want, got)
	}
}
