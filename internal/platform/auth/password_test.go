package auth

import "testing"

func TestHashPasswordAndVerify(t *testing.T) {
	password := "123456"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if !ok {
		t.Fatal("password verify failed")
	}
}

func TestVerifyPasswordFailed(t *testing.T) {
	password := "123456"

	hash, _ := HashPassword(password)

	ok, err := VerifyPassword("654321", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if ok {
		t.Fatal("verify should fail")
	}
}
