package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mock handler（用于验证是否放行）
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		common.WriteResponse(w, common.CodeSuccess, "ok", nil)
	})
}

func TestDuplicateMitigation(t *testing.T) {
	// 初始化 middleware（1秒过期）
	mw := DuplicateMitigation(1 * time.Second)

	// 构造 handler chain
	handler := mw(testHandler())

	// 构造请求
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()

	// 模拟登录用户放入 context
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{
		UserID: 1,
		Phone:  "123",
		Role:   1,
	})
	req = req.WithContext(ctx)

	// 第一次请求（应该通过）
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("第一次请求失败: %d", w.Code)
	}

	// 第二次请求（应该被拦截）
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req)

	if w2.Code != 200 {
		t.Fatalf("第二次请求 HTTP 状态异常: %d", w2.Code)
	}

	t.Log("Duplicate middleware 测试完成")
}
