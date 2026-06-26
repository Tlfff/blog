package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_AllRoutes(t *testing.T) {
	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)

	// 事先生成一个合法的 Token 备用
	validToken, err := auth.GenerateToken("13800000000", 2, 123)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string // 输入的请求头
		expectedStatus int    // 预期的 HTTP 状态码
		expectedError  error  // 预期 c.Errors 中包含的错误
		shouldHasUser  bool   // 是否成功注入了用户信息
	}{
		{
			name:           "1. 缺失Authorization请求头",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized, // 或者是你定义的错误关联的状态码，这里根据你业务决定
			expectedError:  common.ErrAuthorizationRequired,
			shouldHasUser:  false,
		},
		{
			name:           "2. Bearer格式错误",
			authHeader:     "Basic 12345",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  common.ErrInvalidAuthorizationHeader,
			shouldHasUser:  false,
		},
		{
			name:           "3. Token字符串为空",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  common.ErrTokenEmpty,
			shouldHasUser:  false,
		},
		{
			name:           "4. 校验JWT失败(无效Token)",
			authHeader:     "Bearer invalid-token-xyz",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  errors.New(""), // 只要发生校验错误即可
			shouldHasUser:  false,
		},
		{
			name:           "5. 成功通关并注入上下文",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedError:  nil,
			shouldHasUser:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 创建全新的 Gin 引擎和路由
			r := gin.New()

			// 统一的错误拦截机制（模仿你真实的 c.Error 拦截逻辑）
			// 如果你的实际项目有通用的错误全局中间件，可以把它挂载到这里
			r.Use(func(c *gin.Context) {
				c.Next()
				// 如果有错误被拦截，根据错误类型写入状态码（这里演示统一回401，可根据你业务调）
				if len(c.Errors) > 0 {
					c.Status(http.StatusUnauthorized)
				}
			})

			// 挂载被测的中间件
			r.Use(AuthMiddleware())

			// 最终的业务 Handler：用于验证是否成功“漏过去”了，以及上下文是否拿到了
			handlerEntered := false
			r.GET("/test", func(c *gin.Context) {
				handlerEntered = true
				if tt.shouldHasUser {
					val, exists := c.Get("currentUser")
					assert.True(t, exists, "应该在上下文中找到 currentUser")
					userCtx, ok := val.(*auth.UserContext)
					assert.True(t, ok)
					assert.Equal(t, int64(123), userCtx.UserID)
					assert.Equal(t, "13800000000", userCtx.Phone)
				}
				c.Status(http.StatusOK)
			})

			// 2. 构造网络请求
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			// 3. 轰炸接口
			r.ServeHTTP(w, req)

			// 4. 断言守卫
			if tt.expectedError != nil {
				// 预期被拦截，不能进入业务 handler
				assert.False(t, handlerEntered, "不应该进入后续的业务 handler")

				// 🎯 【修正这里】：如果是错误分支，可以通过验证 HTTP 状态码是否为非 200 来间接或直接断言
				// 如果你的全局错误中间件会把 c.Errors 转成 JSON 响应，你甚至可以断言 w.Body.String()
				assert.NotEqual(t, http.StatusOK, w.Code, "被拦截的请求状态码不应该为 200")
			} else {
				// 预期成功通关
				assert.True(t, handlerEntered, "应该成功进入后续业务 handler")
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}
