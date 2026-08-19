package middleware

import (
	"blog/internal/shared/common"
	"log"

	"github.com/gin-gonic/gin"
)

// 统一收口 Handler 通过 c.Error 抛出的错误，转换为业务错误码并返回
func GlobalErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 先执行后续 Handler，等业务处理完再统一处理错误
		c.Next()
		// 业务函数执行完后检查是否有错误
		if len(c.Errors) > 0 {

			// 最后的错误往往是最被业务需要的
			err := c.Errors.Last().Err
			log.Printf("[error] %s %s | 原因: %v\n",
				c.Request.Method, c.Request.URL.Path, err)
			// 2. 将业务错误映射为对外错误码后返回
			bizCode := common.GetCodeByError(err)
			common.Fail(c, bizCode, err.Error())
		}

	}
}
