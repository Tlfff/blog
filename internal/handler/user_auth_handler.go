package handler

import (
	"time"

	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/user"
	userDto "blog/internal/dto/user"
	"blog/internal/model"
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
	// 1. 解析请求体并放进req
	err := c.ShouldBind(&req)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	// 2. 创建新用户
	user := &model.User{
		ID:         0,
		Nickname:   req.Nickname,
		Phone:      req.Phone,
		Avatar:     "https://example.com/default-avatar.png",
		Role:       int8(model.RoleUser),
		Status:     1,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	}
	// 3. 调用服务层进行注册
	err = h.userAuth.Register(user, req.Password)
	if err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "注册成功", nil)
}

// Login 处理用户登录请求
func (h *UserAuthHandler) Login(c *gin.Context) {
	var req user.LoginRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	// 2. 验证用户登录
	dbUser, err := h.userAuth.Login(req.Phone, req.Password)
	if err != nil {

		c.Error(err)
		return
	}

	// 3. 生成JWT令牌
	token, err := auth.GenerateToken(dbUser.Phone, dbUser.Role, dbUser.ID)
	if err != nil {
		c.Error(common.ErrSystem)
		return
	}
	// 4. 封装返回体
	res := userDto.LoginResponse{
		AccessToken: token,
	}
	common.OK(c, "登录成功", res)

}
