package auth

import "testing"

func TestHashPasswordVerifies(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !VerifyPassword("correct horse battery staple", hash) {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword("wrong password", hash) {
		t.Fatal("wrong password verified")
	}
}

func TestVerifyPasswordRejectsInvalidHash(t *testing.T) {
	if VerifyPassword("password", "not-a-real-hash") {
		t.Fatal("invalid hash verified")
	}
}
