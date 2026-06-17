package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"blog/internal/auth"
)

func TestAuthMiddleware_NoHeader(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应该进入 handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestAuthMiddleware_WithContext(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserContext(r.Context())
		if !ok || user == nil {
			t.Fatalf("context 没写入 user")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// 手动塞一个 context（模拟登录态）
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{
		UserID: 1,
		Phone:  "13800000000",
		Role:   1,
	})

	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}
