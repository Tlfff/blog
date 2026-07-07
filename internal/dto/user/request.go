package user

// 用户注册
type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=6"` // 限制密码最少6位
}

// 用户登录
type LoginRequest struct {
	Phone    string `json:"account" binding:"omitempty,numeric"`          // 只能是纯数字
	Nickname string `json:"nickname" binding:"omitempty,not_only_number"` // 不能是纯数字
	Password string `json:"password" binding:"required"`
}

// 更新用户基本信息
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Avatar   string `json:"avatar"`
}

// 更改密码
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
