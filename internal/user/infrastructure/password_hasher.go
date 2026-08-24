package infrastructure

import (
	platformsecurity "blog/internal/platform/security"
	userapp "blog/internal/user/application"
)

type passwordHasher struct{}

// NewPasswordHasher 创建 PBKDF2 密码技术适配器。
func NewPasswordHasher() userapp.PasswordHasher {
	return passwordHasher{}
}

// Hash 生成兼容的 PBKDF2 密码哈希。
func (passwordHasher) Hash(password string) (string, error) {
	return platformsecurity.HashPassword(password)
}

// Verify 校验明文密码和已存储哈希。
func (passwordHasher) Verify(password, storedPassword string) (bool, error) {
	return platformsecurity.VerifyPassword(password, storedPassword)
}
