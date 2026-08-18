package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	userDto "blog/internal/dto/user"
	"context"

	"github.com/gin-gonic/gin"
)

// UserUsecase 是用户资料、改密、账号与头像的应用用例接口。
type UserUsecase interface {
	GetMyProfile(ctx context.Context, userID uint64) (*userDto.MyProfileResponse, error)
	GetUserProfile(ctx context.Context, userID uint64) (*userDto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uint64, nickname, avatar string) error
	VerifyOldPassword(ctx context.Context, userID uint64, oldPassword string) (string, error)
	UpdatePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error
	ChangePassword(ctx context.Context, userID uint64, changeToken, newPassword, currentToken string) error
	UpdateAccount(ctx context.Context, userID uint64, phone string) error
	GetAvatarUploadURL(ctx context.Context, userID uint64, fileExt string) (uploadURL, objectKey string, err error)
	ConfirmAvatar(ctx context.Context, userID uint64, objectKey string) (avatarURL string, err error)
}

type UserHandler struct {
	user     UserUsecase
	userAuth UserAuthUsecase
}

func NewUserHandler(user UserUsecase, userAuth UserAuthUsecase) *UserHandler {
	return &UserHandler{user: user, userAuth: userAuth}
}

// 获取个人资料
func (h *UserHandler) GetMyProfile(c *gin.Context) {
	// 1. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 2. 获取用户个人资料
	res, err := h.user.GetMyProfile(c, userCtx.UserID)
	if err != nil {

		c.Error(err)
		return
	}
	common.OK(c, "获取成功", res)
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
	res, err := h.user.GetUserProfile(c, req.UserId)
	if err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "获取成功", res)
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
	err := h.user.UpdateProfile(c, userCtx.UserID, req.Nickname, req.Avatar)
	if err != nil {

		c.Error(err)
		return
	}

	common.OK(c, "个人资料修改成功", nil)
}

// 验证旧密码并获取一次性修改凭证
func (h *UserHandler) VerifyOldPassword(c *gin.Context) {
	var req userDto.VerifyPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	changeToken, err := h.user.VerifyOldPassword(c, userCtx.UserID, req.OldPassword)
	if err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "旧密码验证成功", gin.H{"change_token": changeToken})
}

// 兼容旧测试和内部调用，HTTP 路由不再使用
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var req userDto.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}
	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	if err := h.user.UpdatePassword(c, userCtx.UserID, req.OldPassword, req.NewPassword); err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "密码修改成功", nil)
}

// 使用一次性修改凭证修改密码
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req userDto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	currentToken := c.GetString("currentToken")
	if err := h.user.ChangePassword(c, userCtx.UserID, req.ChangeToken, req.NewPassword, currentToken); err != nil {
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
	err := h.user.UpdateAccount(c, userCtx.UserID, req.Phone)
	if err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "电话修改成功", nil)
}

// 获取头像上传凭证
func (h *UserHandler) GetAvatarUploadURL(c *gin.Context) {
	var req userDto.GetAvatarUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	uploadURL, objectKey, err := h.user.GetAvatarUploadURL(c, userCtx.UserID, req.FileExt)
	if err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "获取成功", gin.H{
		"upload_url": uploadURL,
		"object_key": objectKey,
	})
}

// 确认头像上传完成
func (h *UserHandler) ConfirmAvatar(c *gin.Context) {
	var req userDto.ConfirmAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	avatarURL, err := h.user.ConfirmAvatar(c, userCtx.UserID, req.ObjectKey)
	if err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "头像更新成功", gin.H{"avatar_url": avatarURL})
}

// 退出登录
func (h *UserHandler) Logout(c *gin.Context) {
	token := c.GetString("currentToken")
	if token == "" {
		c.Error(common.ErrTokenEmpty)
		return
	}
	if err := h.userAuth.Logout(c, token); err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "退出成功", nil)
}
