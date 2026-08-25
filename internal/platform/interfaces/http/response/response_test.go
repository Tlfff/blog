package response

import (
	apperrors "blog/internal/shared/apperrors"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestResponseContract 验证统一成功和失败响应结构。
func TestResponseContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(writer)
	OK(ctx, "获取成功", map[string]any{"id": 1})
	if writer.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码错误: %d", writer.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(writer.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应 JSON 非法: %v", err)
	}
	if body["success"] != true || body["code"] != float64(CodeSuccess) {
		t.Fatalf("成功响应结构发生变化: %s", writer.Body.String())
	}
	if CodeByError(apperrors.ErrUserNotFound) != CodeUserNotFound {
		t.Fatal("用户不存在错误码映射发生变化")
	}
}
