package http

import (
	"blog/internal/platform/interfaces/http/response"
	apperrors "blog/internal/shared/apperrors"
	userresult "blog/internal/user/app/dto"
	"context"

	"github.com/gin-gonic/gin"
)

// UserAuthUsecase 是注册、登录、退出登录的应用用例接口。
type UserAuthUsecase interface {
	// Register 注册用户。
	Register(ctx context.Context, phone, password, nickname, clientIP string) error
	// Login 登录用户并创建会话。
	Login(ctx context.Context, phone, nickname, password, clientIP, device string, rememberMe bool) (*userresult.LoginResponse, error)
	// Logout 删除当前用户会话。
	Logout(ctx context.Context, token string) error
}

// UserAuthHandler 处理用户注册和登录 HTTP 请求。
type UserAuthHandler struct {
	userAuth UserAuthUsecase // 用户认证应用用例
}

// NewUserAuthHandler 创建 User 认证 HTTP Handler。
func NewUserAuthHandler(userAuth UserAuthUsecase) *UserAuthHandler {
	return &UserAuthHandler{userAuth: userAuth}
}

// Register 处理用户注册请求。
func (h *UserAuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	// 1. 解析前端传来的 JSON 请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	// 2. 调用服务层进行注册
	err := h.userAuth.Register(
		c,
		req.Phone,
		req.Password,
		req.Nickname,
		c.ClientIP(),
	)
	if err != nil {
		c.Error(err) // 错误直接交给 Gin 的错误处理中间件
		return
	}

	// 3. 返回成功响应
	response.OK(c, "注册成功", nil)
}

// Login 处理用户登录请求。
func (h *UserAuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	// 1. 解析 JSON 请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}
	// 1.1 两个不能同时为空
	if req.Nickname == "" && req.Phone == "" {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	// 2. 调用登录
	res, err := h.userAuth.Login(c, req.Phone, req.Nickname, req.Password, c.ClientIP(), req.Device, req.RememberMe)
	if err != nil {
		c.Error(err)
		return
	}

	// 3. 将 Service 已经封装好的包含 Token 的 res 完美吐给前端
	response.OK(c, "登录成功", res)

}
