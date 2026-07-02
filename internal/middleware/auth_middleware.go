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
