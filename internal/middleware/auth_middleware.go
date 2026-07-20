package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"strings"

	"github.com/gin-gonic/gin"
)

func checkToken(c *gin.Context) (*auth.UserContext, error) {
	// 1. 校验Authorization请求头
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, common.ErrAuthorizationRequired
	}
	// 2. Bearer 格式校验
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, common.ErrInvalidAuthorizationHeader
	}
	// 3. 截取Token字符串
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, common.ErrTokenEmpty
	}
	// 4. 校验JWT
	claims, err := auth.VerifyToken(token)
	if err != nil {
		return nil, err
	}
	// 5. 组装用户信息并注入Gin上下文
	userCtx := &auth.UserContext{UserID: claims.UserID, Phone: claims.Phone, Role: claims.Role}
	return userCtx, nil
}

// 必须登录的场景
func MustAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, err := checkToken(c)
		// 未登录直接拦截
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		c.Set("currentUser", userCtx)
		c.Next()
	}

}

// 可选认证中间件，登录和未登录有不同逻辑
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, err := checkToken(c)
		// 如果登录了就记录用户信息
		if err == nil {
			c.Set("currentUser", userCtx)
		}
		c.Next()
	}
}
