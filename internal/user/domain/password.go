package domain

import "unicode/utf8"

const (
	// MinimumPasswordLength 表示现有 HTTP 契约要求的最小密码长度。
	MinimumPasswordLength = 6
)

// PlainPassword 表示创建新密码时使用的明文密码值对象。
type PlainPassword struct {
	value string // 明文密码，仅在当前用例内短暂使用
}

// NewPlainPassword 创建符合当前最小长度规则的明文密码。
func NewPlainPassword(value string) (PlainPassword, error) {
	if utf8.RuneCountInString(value) < MinimumPasswordLength {
		return PlainPassword{}, ErrPasswordTooShort
	}
	return PlainPassword{value: value}, nil
}

// String 返回 PasswordHasher 使用的明文密码。
func (p PlainPassword) String() string {
	return p.value
}

// PasswordHash 表示不可变的已哈希密码值对象。
type PasswordHash struct {
	value string // 兼容现有 PBKDF2 存储格式的哈希文本
}

// RestorePasswordHash 从持久化数据重建密码哈希，不增加新的格式限制。
func RestorePasswordHash(value string) PasswordHash {
	return PasswordHash{value: value}
}

// String 返回持久化和密码校验所需的哈希文本。
func (p PasswordHash) String() string {
	return p.value
}
