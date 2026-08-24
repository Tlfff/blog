package auth

import "testing"

// TestHashPasswordAndVerify 验证密码哈希和校验成功路径。
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

// TestVerifyPasswordFailed 验证错误密码校验失败路径。
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
