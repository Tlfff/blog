package handler

import (
	"blog/internal/common"
	"blog/internal/dto/user"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
)

type UserAuthHandler struct {
	userAuth *service.UserAuthService
}

func NewUserAuthHandler(userAuth *service.UserAuthService) *UserAuthHandler {
	return &UserAuthHandler{userAuth: userAuth}
}

// Register 处理用户注册请求
func (h *UserAuthHandler) Register(c *gin.Context) {
	var req user.RegisterRequest
	// 1. 解析前端传来的 JSON 请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
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
	common.OK(c, "注册成功", nil)
}

// Login 处理用户登录请求
func (h *UserAuthHandler) Login(c *gin.Context) {
	var req user.LoginRequest

	// 1. 解析 JSON 请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}
	// 1.1 两个不能同时为空
	if req.Nickname == "" && req.Phone == "" {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	// 2. 调用登录
	res, err := h.userAuth.Login(c, req.Phone, req.Nickname, req.Password, c.ClientIP())
	if err != nil {
		c.Error(err)
		return
	}

	// 3. 将 Service 已经封装好的包含 Token 的 res 完美吐给前端
	common.OK(c, "登录成功", res)

}
