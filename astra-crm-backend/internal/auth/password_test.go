package auth

import "testing"

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "secret" {
		t.Fatal("password hash equals plaintext password")
	}

	if !CheckPassword("secret", hash) {
		t.Fatal("CheckPassword() = false, want true")
	}

	if CheckPassword("wrong", hash) {
		t.Fatal("CheckPassword() = true for wrong password")
	}
}
