package user

// 用户注册
type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=6"` // 限制密码最少6位
}

// func (r *RegisterRequest) Validate() error {
// 	if r.Phone == "" || r.Password == "" || r.Nickname == "" {
// 		return common.ErrRegisterInputEmpty
// 	}
// 	if len(r.Password) < 6 {
// 		return common.ErrPasswordTooShort
// 	}
// 	// 校验角色合法性
// 	if r.Role == 0 || model.FindRoleById(int(r.Role)) != nil {
// 		return common.ErrRoleInvalid
// 	}
// 	return nil
// }

// 用户登录
type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// func (r *LoginRequest) Validate() error {
// 	if r.Phone == "" || r.Password == "" {
// 		return common.ErrLoginInputEmpty
// 	}
// 	return nil
// }

// 更新用户基本信息
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Avatar   string `json:"avatar"`
}

// func (r *UpdateProfileRequest) Validate() error {
// 	if r.Nickname == "" {
// 		return common.ErrNickNameNotFound
// 	}
// 	return nil
// }

// 变更敏感账号信息
type UpdateAccountRequest struct {
	Phone       string `json:"phone" binding:"required"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// func (r *UpdateAccountRequest) Validate() error {
// 	if r.Phone == "" || r.OldPassword == "" || r.NewPassword == "" {
// 		return common.ErrLoginInputEmpty
// 	}
// 	if len(r.NewPassword) < 6 {
// 		return common.ErrPasswordTooShort
// 	}
// 	return nil
// }
