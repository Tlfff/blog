package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 是所有 HTTP 接口的统一响应结构。
type Response struct {
	Success bool   `json:"success"` // 请求是否成功
	Code    int    `json:"code"`    // 业务响应码，取值见 code.go
	Message string `json:"message"` // 提示文案，可直接展示给用户
	Data    any    `json:"data"`    // 响应数据，失败时为 null
}

// OK 返回统一成功响应，HTTP 状态码固定为 200。
func OK(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Code:    200,
		Message: message,
		Data:    data,
	})
}

// Fail 返回统一失败响应，HTTP 状态码仍为 200。
func Fail(c *gin.Context, bizCode int, message string) {
	c.JSON(http.StatusOK, Response{
		Success: false,
		Code:    bizCode,
		Message: message,
		Data:    nil,
	})
}
