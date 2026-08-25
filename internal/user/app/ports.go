package app

import userdomain "blog/internal/user/domain"

// PasswordHasher 定义 User 用例需要的密码哈希与校验能力。
type PasswordHasher interface {
	// Hash 生成兼容的密码存储哈希。
	Hash(password userdomain.PlainPassword) (userdomain.PasswordHash, error)
	// Verify 校验明文密码和已存储哈希。
	Verify(password string, storedPassword userdomain.PasswordHash) (bool, error)
}
