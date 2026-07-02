package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/model"

	"github.com/gin-gonic/gin"
)

// 检查是否为管理者
func AdminCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("currentUser").(*auth.UserContext)
		if user.Role != int8(model.RoleAdmin) {
			// 权限不足
			c.Error(common.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
