package middleware

import (
	"blog/internal/common"
	"log"

	"github.com/gin-gonic/gin"
)

func GlobalErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// 业务函数执行完后检查是否有错误
		if len(c.Errors) > 0 {

			// 最后的错误往往是最被业务需要的
			err := c.Errors.Last().Err
			log.Printf("[error] %s %s | 原因: %v\n",
				c.Request.Method, c.Request.URL.Path, err)
			bizCode := common.GetCodeByError(err)
			common.Fail(c, bizCode, err.Error())
		}

	}
}
