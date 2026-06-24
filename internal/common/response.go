package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

//	func WriteResponse(c *gin.Context, code int, message string, date any) {
//		w.Header().Set("Content-Type", "application/json")
//		_ = json.NewEncoder(w).Encode(Response{Code: code, Message: message, Data: date})
//	}
func OK(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Code:    200,
		Message: message,
		Data:    data,
	})
}
func Fail(c *gin.Context, bizCode int, message string) {
	c.JSON(http.StatusOK, Response{
		Success: false,
		Code:    bizCode,
		Message: message,
		Data:    nil,
	})
}
