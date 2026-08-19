package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponseContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		OK(c, "获取成功", map[string]any{"id": 1})

		if w.Code != http.StatusOK {
			t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if body["success"] != true || body["code"] != float64(200) || body["message"] != "获取成功" {
			t.Fatalf("统一成功响应结构被改变: %s", w.Body.String())
		}
		if _, ok := body["data"]; !ok {
			t.Fatalf("成功响应缺少 data 字段: %s", w.Body.String())
		}
	})

	t.Run("failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Fail(c, CodeUserNotFound, "用户不存在")

		if w.Code != http.StatusOK {
			t.Fatalf("HTTP status = %d, want %d", w.Code, http.StatusOK)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if body["success"] != false || body["code"] != float64(CodeUserNotFound) || body["data"] != nil {
			t.Fatalf("统一失败响应结构被改变: %s", w.Body.String())
		}
	})
}

func TestErrorCodeMappingContract(t *testing.T) {
	tests := []struct {
		err  error
		code int
	}{
		{ErrInvalidRequestBody, CodeBadRequestFormat},
		{ErrParameter, CodeInvalidParameter},
		{ErrAuthorizationRequired, CodeUnauthorized},
		{ErrForbidden, CodeForbidden},
		{ErrDuplicateSubmission, CodeDuplicateSubmission},
		{ErrUserExists, CodeUserExists},
		{ErrUserNotFound, CodeUserNotFound},
		{ErrPasswordFailed, CodePasswordFailed},
		{ErrTokenInvalid, CodeTokenInvalid},
		{ErrTokenExpired, CodeTokenExpired},
		{ErrArticleNotFound, CodeArticleNotFound},
		{ErrArticlePermissionDenied, CodeArticlePermission},
		{ErrCommentNotFound, CodeCommentNotFound},
		{ErrCommentPermission, CodeCommentPermission},
		{ErrLockExpired, CodeLockExpired},
		{ErrKafkaSendFailed, CodeKafkaSendFailed},
	}

	for _, tt := range tests {
		if got := GetCodeByError(tt.err); got != tt.code {
			t.Errorf("GetCodeByError(%v) = %d, want %d", tt.err, got, tt.code)
		}
	}

	if got := GetCodeByError(nil); got != CodeInternalServerError {
		t.Errorf("未知错误映射 = %d, want %d", got, CodeInternalServerError)
	}
}
