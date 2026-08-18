package identity

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const (
	PasswordHashAlgorithm = "pbkdf2"
	HashIterations        = 100000
	HashKeyLength         = 32
	SaltLength            = 16
)

// HashPassword 生成 PBKDF2 密码哈希。
func HashPassword(password string) (string, error) {
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	saltStr := hex.EncodeToString(salt)
	hash, err := pbkdf2.Key(sha256.New, password, salt, HashIterations, HashKeyLength)
	if err != nil {
		return "", err
	}
	hashStr := hex.EncodeToString(hash)
	return fmt.Sprintf("%s$%d$%s$%s", PasswordHashAlgorithm, HashIterations, saltStr, hashStr), nil
}

// VerifyPassword 校验密码与存储哈希是否匹配。
func VerifyPassword(password, storedPassword string) (bool, error) {
	parts := strings.Split(storedPassword, "$")
	if len(parts) != 4 {
		return false, fmt.Errorf("无效的存储密码格式")
	}
	algorithm, iterationsStr, saltStr, hashStr := parts[0], parts[1], parts[2], parts[3]
	if algorithm != PasswordHashAlgorithm {
		return false, fmt.Errorf("不支持的哈希算法")
	}
	iterations, err := strconv.Atoi(iterationsStr)
	if err != nil {
		return false, fmt.Errorf("无效的迭代次数")
	}
	salt, err := hex.DecodeString(saltStr)
	if err != nil {
		return false, fmt.Errorf("无效的盐值")
	}
	storedHash, err := hex.DecodeString(hashStr)
	if err != nil || len(storedHash) == 0 {
		return false, fmt.Errorf("无效的哈希值")
	}
	inputHash, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(storedHash))
	if err != nil {
		return false, fmt.Errorf("生成输入密码哈希时出错")
	}
	return hmac.Equal(inputHash, storedHash), nil
}
