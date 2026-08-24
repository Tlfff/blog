package auth

import "testing"

// TestHMACSignAndVerify 验证 HMAC 签名和篡改检测。
func TestHMACSignAndVerify(t *testing.T) {
	secretKey := "sk-partner-a-secret-key"
	accessKeyID := "ak-partner-a"
	methodName := "/blogopen.v1.UserService/GetPublicUserInfo"
	timestamp := "1700000000"
	nonce := "abc123"
	bodyHash := BuildBodyHash([]byte{0x08, 0x01}) // 序列化后的请求体

	// 1. 正确密钥应通过
	sig := HMACSign(secretKey, accessKeyID, methodName, timestamp, nonce, bodyHash)
	if !HMACVerify(secretKey, accessKeyID, methodName, timestamp, nonce, bodyHash, sig) {
		t.Error("正确签名应通过校验")
	}

	// 2. 错误密钥应拒绝
	if HMACVerify("wrong-secret", accessKeyID, methodName, timestamp, nonce, bodyHash, sig) {
		t.Error("错误密钥不应通过校验")
	}

	// 3. 篡改签名原文任一字段应拒绝
	if HMACVerify(secretKey, accessKeyID, methodName, timestamp, "another-nonce", bodyHash, sig) {
		t.Error("篡改nonce不应通过校验")
	}
	if HMACVerify(secretKey, accessKeyID, "/blogopen.v1.UserService/GetUserBasicInfo", timestamp, nonce, bodyHash, sig) {
		t.Error("篡改method_name不应通过校验")
	}
	if HMACVerify(secretKey, accessKeyID, methodName, timestamp, nonce, "different-body-hash", sig) {
		t.Error("篡改请求体不应通过校验")
	}
}

// TestBuildHMACStringToSign 验证 HMAC 待签名原文格式。
func TestBuildHMACStringToSign(t *testing.T) {
	bodyHash := BuildBodyHash([]byte{0x08, 0x01})
	want := "ak-partner-a\n/blogopen.v1.UserService/GetPublicUserInfo\n1700000000\nabc123\n" + bodyHash
	if got := BuildHMACStringToSign("ak-partner-a", "/blogopen.v1.UserService/GetPublicUserInfo", "1700000000", "abc123", bodyHash); got != want {
		t.Errorf("签名原文拼装错误: got %q want %q", got, want)
	}
}

// TestBuildBodyHash 验证请求体哈希结果。
func TestBuildBodyHash(t *testing.T) {
	// 同一份请求体哈希应稳定一致
	h1 := BuildBodyHash([]byte("hello"))
	h2 := BuildBodyHash([]byte("hello"))
	if h1 != h2 {
		t.Errorf("相同请求体哈希应一致: %q vs %q", h1, h2)
	}
	if h1 == BuildBodyHash([]byte("hello!")) {
		t.Error("不同请求体哈希不应一致")
	}
}
