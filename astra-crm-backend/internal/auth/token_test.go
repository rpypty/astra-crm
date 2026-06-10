package auth

import "testing"

func TestSessionTokenHashDoesNotExposeRawToken(t *testing.T) {
	token, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken() error = %v", err)
	}

	hash := HashSessionToken(token)
	if hash == token {
		t.Fatal("token hash equals raw token")
	}

	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64 hex chars", len(hash))
	}
}
