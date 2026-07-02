package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 校验Authorization请求头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(common.ErrAuthorizationRequired)
			c.Abort()
			return
		}
		// 2. Bearer 格式校验
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Error(common.ErrInvalidAuthorizationHeader)
			c.Abort()
			return
		}
		// 3. 截取Token字符串
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.Error(common.ErrTokenEmpty)
			c.Abort()
			return
		}
		// 4. 校验JWT
		claims, err := auth.VerifyToken(token)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		// 5. 组装用户信息并注入Gin上下文
		userCtx := &auth.UserContext{UserID: claims.UserID, Phone: claims.Phone, Role: claims.Role}
		c.Set("currentUser", userCtx)

		c.Next()
	}

}

// 可选认证中间件，如果没登录也能访问，登录了就注入用户id
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		// 1. 没带 Token，直接放行给下一个 Handler，此时当做游客
		if authHeader == "" {
			c.Next()
			return
		}
		// 2. Bearer 格式校验
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}
		// 3. 截取Token字符串
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.Next()
			return
		}

		// 4. 校验JWT
		claims, err := auth.VerifyToken(token)
		if err != nil {
			c.Next() // Token 解析失败（比如过期了）依然不拦截，放行
			return
		}

		// 5. 解析成功，把 userID 塞进上下文
		c.Set("userID", claims.UserID)
		c.Next()
	}
}
