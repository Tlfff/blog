package infra

import (
	platformsecurity "blog/internal/platform/security"
	userapp "blog/internal/user/app"
	userdomain "blog/internal/user/domain"
)

// passwordHasher 将 Platform Security 适配为 User PasswordHasher Port。
type passwordHasher struct{}

// NewPasswordHasher 创建 PBKDF2 密码技术适配器。
func NewPasswordHasher() userapp.PasswordHasher {
	// 1. 返回无状态密码哈希适配器
	return passwordHasher{}
}

// Hash 生成兼容的 PBKDF2 密码哈希。
func (passwordHasher) Hash(password userdomain.PlainPassword) (userdomain.PasswordHash, error) {
	// 1. 调用 Platform Security 生成兼容哈希
	hash, err := platformsecurity.HashPassword(password.String())
	if err != nil {
		return userdomain.PasswordHash{}, err
	}
	return userdomain.RestorePasswordHash(hash), nil
}

// Verify 校验明文密码和已存储哈希。
func (passwordHasher) Verify(password string, storedPassword userdomain.PasswordHash) (bool, error) {
	// 1. 调用 Platform Security 校验密码
	return platformsecurity.VerifyPassword(password, storedPassword.String())
}
