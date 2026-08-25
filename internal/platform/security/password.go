package auth

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
	PasswordHashAlgorithm = "pbkdf2" // 密码哈希算法标识
	HashIteratiaons       = 100000   // PBKDF2 迭代次数
	HashKeyLength         = 32       // 派生密钥长度，单位：字节
	SaltLength            = 16       // 随机盐长度，单位：字节
)

// HashPassword 生成 PBKDF2 密码哈希。
func HashPassword(password string) (string, error) {
	// 1. 生成随机盐
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// 2. 使用 PBKDF2-SHA256 派生密钥
	hash, err := pbkdf2.Key(sha256.New, password, salt, HashIteratiaons, HashKeyLength)
	if err != nil {
		return "", err
	}

	// 3. 拼装兼容存储格式
	return fmt.Sprintf("%s$%d$%s$%s", PasswordHashAlgorithm, HashIteratiaons, hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

// VerifyPassword 校验明文密码与已存储哈希是否匹配。
func VerifyPassword(password, storedPassword string) (bool, error) {
	// 1. 拆解兼容存储格式
	parts := strings.Split(storedPassword, "$")
	if len(parts) != 4 {
		return false, fmt.Errorf("无效的存储密码格式")
	}
	if parts[0] != PasswordHashAlgorithm {
		return false, fmt.Errorf("不支持的哈希算法")
	}

	// 2. 解析哈希参数
	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("无效的迭代次数")
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false, fmt.Errorf("无效的盐值")
	}
	storedHash, err := hex.DecodeString(parts[3])
	if err != nil || len(storedHash) == 0 {
		return false, fmt.Errorf("无效的哈希值")
	}

	// 3. 重新计算并使用恒定时间比较
	inputHash, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(storedHash))
	if err != nil {
		return false, fmt.Errorf("生成输入密码哈希时出错")
	}
	return hmac.Equal(inputHash, storedHash), nil
}
