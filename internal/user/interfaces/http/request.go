package http

// 用户注册
type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=6"` // 限制密码最少6位
}

// 用户登录
type LoginRequest struct {
	Phone      string `json:"phone" binding:"omitempty,numeric"`            // 只能是纯数字
	Nickname   string `json:"nickname" binding:"omitempty,not_only_number"` // 不能是纯数字
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"` // 记住我，延长token有效期
	Device     string `json:"device"`      // 设备标识，如 web/ios/android
}

// 更新用户基本信息
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Avatar   string `json:"avatar"`
}

// 验证旧密码
type VerifyPasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
}

// 修改密码
type ChangePasswordRequest struct {
	ChangeToken string `json:"change_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// 兼容旧测试和内部调用，HTTP 路由不再使用
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// 变更敏感账号信息-电话
type UpdateAccountRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// 查看他人主页
type GetPublicProfileRequest struct {
	UserId uint64 `form:"user_id" binding:"required"`
}

// 获取头像上传凭证
type GetAvatarUploadURLRequest struct {
	FileExt string `json:"file_ext" binding:"required"`
}

// 确认头像上传完成
type ConfirmAvatarRequest struct {
	ObjectKey string `json:"object_key" binding:"required"`
}
