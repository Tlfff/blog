package middleware

import (
	"blog/internal/platform/interfaces/http/response"
	"log"

	"github.com/gin-gonic/gin"
)

// GlobalErrorMiddleware 将 Gin Error 映射为兼容业务响应。
func GlobalErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		log.Printf("[error] %s %s | 原因: %v\n", c.Request.Method, c.Request.URL.Path, err)
		response.Fail(c, response.CodeByError(err), err.Error())
	}
}
