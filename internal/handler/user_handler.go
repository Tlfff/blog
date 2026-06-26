package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	userDto "blog/internal/dto/user"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	user *service.UserService
}

func NewUserHandler(user *service.UserService) *UserHandler {
	return &UserHandler{user: user}
}

// 获取个人资料
func (h *UserHandler) GetMyProfile(c *gin.Context) {
	// 1. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 2. 获取用户个人资料
	user, err := h.user.GetProfile(userCtx.UserID)
	if err != nil {

		c.Error(err)
		return
	}
	common.OK(c, "获取成功", userDto.NewMyProfileResponse(user))
}

// 查看他人主页
func (h *UserHandler) GetPublicProfile(c *gin.Context) {
	// 1.获取用户ID
	var req userDto.GetPublicProfileRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrUserNotFound)
		return
	}

	// 2. 获取用户主页信息
	user, err := h.user.GetProfile(req.UserId)
	if err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "获取成功", userDto.NewUserProfileResponse(user))
}

// 修改基础资料
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// 1. 解析请求体并放进req
	var req userDto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {

		c.Error(common.ErrInvalidRequestBody)
		return
	}

	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 更新资料
	err := h.user.UpdateProfile(userCtx.UserID, req.Nickname, req.Avatar)
	if err != nil {

		c.Error(err)
		return
	}

	common.OK(c, "个人资料修改成功", nil)
}

// 修改密码
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	// 1. 解析请求体并放进req
	var req userDto.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}
	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 更新密码
	err := h.user.UpdatePassword(userCtx.UserID, req.OldPassword, req.NewPassword)
	if err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "密码修改成功", nil)
}

// 修改账户信息（电话）
func (h *UserHandler) UpdateAccount(c *gin.Context) {
	// 1. 解析请求体并放进req
	var req userDto.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}
	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 更新密码
	err := h.user.UpdateAccount(userCtx.UserID, req.Phone)
	if err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "电话修改成功", nil)
}

// 退出登录
// todo :设置黑名单
func (h *UserHandler) Logout(c *gin.Context) {

	common.OK(c, "退出成功", nil)
}
