package http

import (
	"blog/internal/platform/interfaces/http/response"
	"blog/internal/platform/security"
	apperrors "blog/internal/shared/apperrors"
	userresult "blog/internal/user/app/dto"
	"context"

	"github.com/gin-gonic/gin"
)

// UserUsecase 是用户资料、改密、账号与头像的应用用例接口。
type UserUsecase interface {
	// GetMyProfile 查询当前用户完整资料。
	GetMyProfile(ctx context.Context, userID uint64) (*userresult.MyProfileResponse, error)
	// GetUserProfile 查询指定用户公开资料。
	GetUserProfile(ctx context.Context, userID uint64) (*userresult.UserProfileResponse, error)
	// UpdateProfile 更新用户昵称和头像资料。
	UpdateProfile(ctx context.Context, userID uint64, nickname, avatar string) error
	// VerifyOldPassword 校验旧密码并签发改密凭证。
	VerifyOldPassword(ctx context.Context, userID uint64, oldPassword string) (string, error)
	// UpdatePassword 使用旧密码修改密码。
	UpdatePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error
	// ChangePassword 使用一次性凭证修改密码。
	ChangePassword(ctx context.Context, userID uint64, changeToken, newPassword, currentToken string) error
	// UpdateAccount 修改用户手机号。
	UpdateAccount(ctx context.Context, userID uint64, phone string) error
	// GetAvatarUploadURL 获取头像上传凭证。
	GetAvatarUploadURL(ctx context.Context, userID uint64, fileExt string) (uploadURL, objectKey string, err error)
	// ConfirmAvatar 确认头像上传完成。
	ConfirmAvatar(ctx context.Context, userID uint64, objectKey string) (avatarURL string, err error)
}

// UserHandler 处理 User 上下文的资料和账户 HTTP 请求。
type UserHandler struct {
	user     UserUsecase     // 用户资料应用用例
	userAuth UserAuthUsecase // 用户认证应用用例
}

// NewUserHandler 创建 User HTTP Handler。
func NewUserHandler(user UserUsecase, userAuth UserAuthUsecase) *UserHandler {
	return &UserHandler{user: user, userAuth: userAuth}
}

// GetMyProfile 获取当前用户资料。
func (h *UserHandler) GetMyProfile(c *gin.Context) {
	// 1. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 2. 获取用户个人资料
	res, err := h.user.GetMyProfile(c, userCtx.UserID)
	if err != nil {

		c.Error(err)
		return
	}
	response.OK(c, "获取成功", res)
}

// GetPublicProfile 查询指定用户公开资料。
func (h *UserHandler) GetPublicProfile(c *gin.Context) {
	// 1.获取用户ID
	var req GetPublicProfileRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrUserNotFound)
		return
	}

	// 2. 获取用户主页信息
	res, err := h.user.GetUserProfile(c, req.UserID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "获取成功", res)
}

// UpdateProfile 修改当前用户基础资料。
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// 1. 解析请求体并放进req
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {

		c.Error(apperrors.ErrInvalidRequestBody)
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

	response.OK(c, "个人资料修改成功", nil)
}

// VerifyOldPassword 验证旧密码并获取一次性修改凭证。
func (h *UserHandler) VerifyOldPassword(c *gin.Context) {
	var req VerifyPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	changeToken, err := h.user.VerifyOldPassword(c, userCtx.UserID, req.OldPassword)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "旧密码验证成功", gin.H{"change_token": changeToken})
}

// UpdatePassword 兼容内部调用的旧密码修改入口。
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}
	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	if err := h.user.UpdatePassword(c, userCtx.UserID, req.OldPassword, req.NewPassword); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "密码修改成功", nil)
}

// ChangePassword 使用一次性凭证修改密码。
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	currentToken := c.GetString("currentToken")
	if err := h.user.ChangePassword(c, userCtx.UserID, req.ChangeToken, req.NewPassword, currentToken); err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "密码修改成功", nil)
}

// UpdateAccount 修改当前用户手机号。
func (h *UserHandler) UpdateAccount(c *gin.Context) {
	// 1. 解析请求体并放进req
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
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

	response.OK(c, "电话修改成功", nil)
}

// GetAvatarUploadURL 获取头像上传凭证。
func (h *UserHandler) GetAvatarUploadURL(c *gin.Context) {
	var req GetAvatarUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	uploadURL, objectKey, err := h.user.GetAvatarUploadURL(c, userCtx.UserID, req.FileExt)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "获取成功", gin.H{
		"upload_url": uploadURL,
		"object_key": objectKey,
	})
}

// ConfirmAvatar 确认头像上传完成。
func (h *UserHandler) ConfirmAvatar(c *gin.Context) {
	var req ConfirmAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	avatarURL, err := h.user.ConfirmAvatar(c, userCtx.UserID, req.ObjectKey)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "头像更新成功", gin.H{"avatar_url": avatarURL})
}

// Logout 退出当前登录会话。
func (h *UserHandler) Logout(c *gin.Context) {
	token := c.GetString("currentToken")
	if token == "" {
		c.Error(apperrors.ErrTokenEmpty)
		return
	}
	if err := h.userAuth.Logout(c, token); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "退出成功", nil)
}
