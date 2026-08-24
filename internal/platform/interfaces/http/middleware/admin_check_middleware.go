package middleware

import (
	"blog/internal/platform/security"
	apperrors "blog/internal/shared/apperrors"
	"blog/internal/user/domain"

	"github.com/gin-gonic/gin"
)

// AdminCheckMiddleware 创建管理员权限校验中间件。
func AdminCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从上下文取出鉴权中间件注入的用户信息
		user := c.MustGet("currentUser").(*auth.UserContext)
		// 2. 角色不是管理员则中断请求
		if user.Role != domain.RoleAdmin {
			// 权限不足
			c.Error(apperrors.ErrForbidden)
			c.Abort()
			return
		}
		// 3. 校验通过，放行到后续 Handler
		c.Next()
	}
}
