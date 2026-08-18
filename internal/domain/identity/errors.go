package identity

import "errors"

// Identity 领域错误，供 Application 层判断并映射为对外错误码
var (
	ErrUserNotFound        = errors.New("用户不存在")        // 按ID或账号查询用户时未命中
	ErrPasswordChangeToken = errors.New("密码修改凭证无效或已过期") // 一次性改密凭证校验失败
	ErrPhoneAlreadyExists  = errors.New("手机号已被注册")      // 注册或换绑手机号时唯一性冲突
)
