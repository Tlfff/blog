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

// 密码哈希参数
const (
	PasswordHashAlgorithm = "pbkdf2" // 哈希算法标识，写入存储格式的第一段
	HashIterations        = 100000   // PBKDF2 迭代次数
	HashKeyLength         = 32       // 派生密钥长度（字节）
	SaltLength            = 16       // 随机盐长度（字节）
)

// 生成 PBKDF2 密码哈希，返回“算法$迭代次数$Salt$Hash”格式的存储字符串
func HashPassword(password string) (string, error) {
	// 1. 生成随机盐
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	saltStr := hex.EncodeToString(salt)

	// 2. 计算密码哈希
	// 2.1 使用 PBKDF2-SHA256 派生密钥
	hash, err := pbkdf2.Key(sha256.New, password, salt, HashIterations, HashKeyLength)
	if err != nil {
		return "", err
	}
	// 2.2 将派生密钥编码为十六进制字符串
	hashStr := hex.EncodeToString(hash)

	// 3. 拼装最终存储格式：算法$迭代次数$Salt$Hash
	return fmt.Sprintf("%s$%d$%s$%s", PasswordHashAlgorithm, HashIterations, saltStr, hashStr), nil
}

// 校验明文密码与存储哈希是否匹配，storedPassword 需为 HashPassword 生成的格式
func VerifyPassword(password, storedPassword string) (bool, error) {
	// 1. 拆解存储格式并校验字段数量
	parts := strings.Split(storedPassword, "$")
	if len(parts) != 4 {
		return false, fmt.Errorf("无效的存储密码格式")
	}
	algorithm, iterationsStr, saltStr, hashStr := parts[0], parts[1], parts[2], parts[3]

	// 2. 逐项解析哈希参数
	// 2.1 校验算法标识是否受支持
	if algorithm != PasswordHashAlgorithm {
		return false, fmt.Errorf("不支持的哈希算法")
	}
	// 2.2 解析迭代次数
	iterations, err := strconv.Atoi(iterationsStr)
	if err != nil {
		return false, fmt.Errorf("无效的迭代次数")
	}
	// 2.3 解码盐值
	salt, err := hex.DecodeString(saltStr)
	if err != nil {
		return false, fmt.Errorf("无效的盐值")
	}
	// 2.4 解码存储的哈希值
	storedHash, err := hex.DecodeString(hashStr)
	if err != nil || len(storedHash) == 0 {
		return false, fmt.Errorf("无效的哈希值")
	}

	// 3. 用相同参数对输入密码重新派生哈希
	inputHash, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(storedHash))
	if err != nil {
		return false, fmt.Errorf("生成输入密码哈希时出错")
	}

	// 4. 使用恒定时间比较，避免时序攻击泄露信息
	return hmac.Equal(inputHash, storedHash), nil
}
