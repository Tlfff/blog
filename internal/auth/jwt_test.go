package auth

import "testing"

func TestGenerateAndVerifyToken(t *testing.T) {
	// 生成Token
	token, err := GenerateToken("1234567890", 1, 12345)
	if err != nil {
		t.Fatalf("生成Token失败: %v", err)
	}
	t.Logf("生成的Token: %s", token)

	// 验证Token
	claims, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("验证Token失败: %v", err)
	}
	t.Logf("解析后的Claims: %+v", claims)

}
