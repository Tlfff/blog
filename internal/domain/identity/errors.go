package identity

import "errors"

var (
	ErrUserNotFound            = errors.New("用户不存在")
	ErrPasswordChangeToken     = errors.New("密码修改凭证无效或已过期")
	ErrPhoneAlreadyExists      = errors.New("手机号已被注册")
)
