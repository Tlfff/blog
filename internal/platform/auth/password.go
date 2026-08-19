package auth

import identity "blog/internal/user/domain"

// 密码哈希参数，转发自 Identity 领域，保持与领域实现一致
const (
	PasswordHashAlgorithm = identity.PasswordHashAlgorithm // 密码哈希算法名
	HashIteratiaons       = identity.HashIterations        // PBKDF2 迭代次数
	HashKeyLength         = identity.HashKeyLength         // 派生密钥长度（字节）
	SaltLength            = identity.SaltLength            // 随机盐长度（字节）
)

// 生成密码哈希
func HashPassword(password string) (string, error) {
	return identity.HashPassword(password)
}

// 验证密码
func VerifyPassword(password, storedPassword string) (bool, error) {
	return identity.VerifyPassword(password, storedPassword)
}
