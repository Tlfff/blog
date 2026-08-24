package http

import (
	"blog/internal/platform/interfaces/http/response"
	apperrors "blog/internal/shared/apperrors"
	userresult "blog/internal/user/application/dto"
	"context"

	"github.com/gin-gonic/gin"
)

// UserAuthUsecase 是注册、登录、退出登录的应用用例接口。
type UserAuthUsecase interface {
	Register(ctx context.Context, phone, password, nickname, clientIP string) error
	Login(ctx context.Context, phone, nickname, password, clientIP, device string, rememberMe bool) (*userresult.LoginResponse, error)
	Logout(ctx context.Context, token string) error
}

type UserAuthHandler struct {
	userAuth UserAuthUsecase
}

func NewUserAuthHandler(userAuth UserAuthUsecase) *UserAuthHandler {
	return &UserAuthHandler{userAuth: userAuth}
}

// 处理用户注册请求
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

// 处理用户登录请求
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
