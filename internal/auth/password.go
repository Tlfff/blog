package auth

import "blog/internal/domain/identity"

const (
	PasswordHashAlgorithm = identity.PasswordHashAlgorithm
	HashIteratiaons       = identity.HashIterations
	HashKeyLength         = identity.HashKeyLength
	SaltLength            = identity.SaltLength
)

// 生成密码哈希
func HashPassword(password string) (string, error) {
	return identity.HashPassword(password)
}

// 验证密码
func VerifyPassword(password, storedPassword string) (bool, error) {
	return identity.VerifyPassword(password, storedPassword)
}
