package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HMAC 三方服务签名相关常量
const (
	// HMACSignSeparator 签名原文字段分隔符
	HMACSignSeparator = "\n"
)

// BuildBodyHash 计算请求体哈希：SHA256(序列化后的request)，返回十六进制
func BuildBodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// BuildHMACStringToSign 构建三方签名原文：
// access_key_id + method_name + timestamp + nonce + request_body_hash
//
// request_body_hash 为请求体的 SHA256（如 SHA256(proto.Marshal(request))），
// 防止请求参数被篡改。调用方与服务端必须使用相同的序列化规则。
func BuildHMACStringToSign(accessKeyID, methodName, timestamp, nonce, requestBodyHash string) string {
	return strings.Join([]string{accessKeyID, methodName, timestamp, nonce, requestBodyHash}, HMACSignSeparator)
}

// HMACSign 用 secretKey 对签名原文做 HMAC-SHA256，返回十六进制签名
func HMACSign(secretKey, accessKeyID, methodName, timestamp, nonce, requestBodyHash string) string {
	stringToSign := BuildHMACStringToSign(accessKeyID, methodName, timestamp, nonce, requestBodyHash)
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil))
}

// HMACVerify 恒定时间比较签名是否一致，防止时序攻击
func HMACVerify(secretKey, accessKeyID, methodName, timestamp, nonce, requestBodyHash, signature string) bool {
	expected := HMACSign(secretKey, accessKeyID, methodName, timestamp, nonce, requestBodyHash)
	return hmac.Equal([]byte(expected), []byte(signature))
}
