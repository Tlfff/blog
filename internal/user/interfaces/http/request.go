package http

// RegisterRequest 表示用户注册请求。
type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"`       // 用户昵称
	Phone    string `json:"phone" binding:"required"`          // 用户手机号
	Password string `json:"password" binding:"required,min=6"` // 明文密码，最少 6 位
}

// LoginRequest 表示用户登录请求。
type LoginRequest struct {
	Phone      string `json:"phone" binding:"omitempty,numeric"`            // 只能是纯数字
	Nickname   string `json:"nickname" binding:"omitempty,not_only_number"` // 不能是纯数字
	Password   string `json:"password" binding:"required"`                  // 明文密码
	RememberMe bool   `json:"remember_me"`                                  // 是否延长 Token 有效期
	Device     string `json:"device"`                                       // 设备标识，如 web、ios、android
}

// UpdateProfileRequest 表示更新用户资料请求。
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required"` // 新用户昵称
	Avatar   string `json:"avatar"`                      // 新头像地址，可以为空
}

// VerifyPasswordRequest 表示验证旧密码请求。
type VerifyPasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"` // 旧明文密码
}

// ChangePasswordRequest 表示使用一次性凭证修改密码的请求。
type ChangePasswordRequest struct {
	ChangeToken string `json:"change_token" binding:"required"`       // 一次性密码修改凭证
	NewPassword string `json:"new_password" binding:"required,min=6"` // 新明文密码，最少 6 位
}

// UpdatePasswordRequest 表示兼容内部调用的密码修改请求。
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`       // 旧明文密码
	NewPassword string `json:"new_password" binding:"required,min=6"` // 新明文密码，最少 6 位
}

// UpdateAccountRequest 表示修改用户手机号请求。
type UpdateAccountRequest struct {
	Phone string `json:"phone" binding:"required"` // 新手机号
}

// GetPublicProfileRequest 表示查询用户公开资料请求。
type GetPublicProfileRequest struct {
	UserID uint64 `form:"user_id" binding:"required"` // 用户唯一标识
}

// GetAvatarUploadURLRequest 表示获取头像上传凭证请求。
type GetAvatarUploadURLRequest struct {
	FileExt string `json:"file_ext" binding:"required"` // 文件扩展名
}

// ConfirmAvatarRequest 表示确认头像上传完成请求。
type ConfirmAvatarRequest struct {
	ObjectKey string `json:"object_key" binding:"required"` // MinIO 对象 Key
}
