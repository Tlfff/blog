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
		// 1. 从上下文取出鉴权中间件注入的用户信息
		user := c.MustGet("currentUser").(*auth.UserContext)
		// 2. 角色不是管理员则中断请求
		if user.Role != int8(model.RoleAdmin) {
			// 权限不足
			c.Error(common.ErrForbidden)
			c.Abort()
			return
		}
		// 3. 校验通过，放行到后续 Handler
		c.Next()
	}
}
