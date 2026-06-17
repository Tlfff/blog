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
	PasswordHashAlgorithm = "pbkdf2" // 哈希算法
	HashIteratiaons       = 100000   // 哈希迭代次数
	HashKeyLength         = 32       // 最终哈希输出字节长度
	SaltLength            = 16       // 盐值长度（字节）
)

// 生成密码哈希
func HashPassword(password string) (string, error) {
	salt := make([]byte, SaltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	saltStr := hex.EncodeToString(salt) // 将盐值转换为十六进制字符串存储
	// 生成哈希值并返回
	hash, err := pbkdf2.Key(sha256.New, password, salt, HashIteratiaons, HashKeyLength)
	if err != nil {
		return "", err
	}
	hashStr := hex.EncodeToString(hash)
	// 返回格式：算法$迭代次数$盐值$哈希值
	return fmt.Sprintf("%s$%d$%s$%s", PasswordHashAlgorithm, HashIteratiaons, saltStr, hashStr), nil
}

// 验证密码
func VerifyPassword(password, storedPassword string) (bool, error) {
	// 解析存储的密码哈希
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
	// 生成输入密码的哈希并与存储的哈希进行比较
	inputHash, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(storedHash))
	if err != nil {
		return false, fmt.Errorf("生成输入密码哈希时出错")
	}
	return hmac.Equal(inputHash, storedHash), nil
}
