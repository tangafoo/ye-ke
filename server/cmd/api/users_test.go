package main

import (
	"strings"
	"testing"
)

func TestNewToken(t *testing.T) {
	token, hash, err := newToken()
	if err != nil {
		t.Fatalf("newToken: %v", err)
	}

	if !strings.HasPrefix(token, "yk_") {
		t.Errorf("token %q missing yk_ prefix", token)
	}
	if len(token) != 3+64 { // "yk_" + 32 bytes hex
		t.Errorf("token length = %d, want %d", len(token), 3+64)
	}
	if hash != hashToken(token) {
		t.Errorf("returned hash does not match hashToken(token)")
	}
	if hash == token {
		t.Errorf("hash must not equal the token")
	}

	// Two mints must never collide — the token is the whole credential.
	token2, _, err := newToken()
	if err != nil {
		t.Fatalf("newToken second mint: %v", err)
	}
	if token2 == token {
		t.Errorf("two mints produced the same token")
	}
}
