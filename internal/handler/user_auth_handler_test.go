package handler

import (
	"bytes"
	"testing"

	"blog/internal/auth"
	"blog/internal/dto/user"
	"blog/internal/repository"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
)

func TestUserAuthHandler_AllRoutes(t *testing.T) {
	// 1. 🟢 组装真实的内存链路（注册和登录会共享同一个内存 Repo 里的数据）
	userRepo := repository.NewUserRepository()
	userAuthService := service.NewUserAuthService(userRepo)
	h := NewUserAuthHandler(userAuthService)

	// 2. 🎯 大表格：按“时光流逝”的顺序，先测异常，再测成功注册，最后测登录
	tests := []struct {
		name           string
		run            func(c *gin.Context) // 动态调用目标函数
		method         string
		path           string
		body           interface{}
		ctxUser        *auth.UserContext
		expectContains string // 预期返回包含的内容
	}{
		// ==================== 🔐 场景 A：用户注册 (Register) ====================
		{
			name:           "1. 注册-请求体错误(触发第一个if)",
			run:            h.Register,
			method:         "POST",
			path:           "/auth/register",
			body:           "我是坏的JSON字符串",
			ctxUser:        nil,
			expectContains: "", // 触发 c.Error，实际返回空
		},
		{
			name:   "2. 注册-成功通关",
			run:    h.Register,
			method: "POST",
			path:   "/auth/register",
			body: user.RegisterRequest{
				Nickname: "林风",
				Phone:    "18078789119",
				Password: "123456",
			},
			ctxUser:        nil,
			expectContains: `"注册成功"`,
		},

		// ==================== 🔓 场景 B：用户登录 (Login) ====================
		{
			name:           "3. 登录-请求体错误(触发第一个if)",
			run:            h.Login,
			method:         "POST",
			path:           "/auth/login",
			body:           "格式不对的JSON",
			ctxUser:        nil,
			expectContains: "",
		},
		{
			name:   "4. 登录-账号或密码错误(触发Service报错if)",
			run:    h.Login,
			method: "POST",
			path:   "/auth/login",
			body: user.LoginRequest{
				Account:  "18078789119",
				Password: "789321", // 故意输错密码
			},
			ctxUser:        nil,
			expectContains: "", // 会被 c.Error(err) 拦下
		},
		{
			name:   "5. 登录-成功通关(拿到JWT令牌)",
			run:    h.Login,
			method: "POST",
			path:   "/auth/login",
			body: user.LoginRequest{
				Account:  "18078789119", // 使用第2步成功注册的手机号
				Password: "123456",
			},
			ctxUser:        nil,
			expectContains: `"access_token"`, // 成功登录应该能拿到你的令牌字段
		},
	}

	// 3. 🤖 驱动引擎
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用之前你在 user_test.go 里写好的 makeTestContext 工具函数
			c, w := makeTestContext(tt.method, tt.path, tt.body, tt.ctxUser)

			// 轰炸目标接口
			tt.run(c)

			// 调试辅助日志
			actualBody := w.Body.String()
			if actualBody == "" && len(c.Errors) > 0 {
				actualBody = "[被 c.Error 拦截] 原因: " + c.Errors.Last().Error()
			}

			// 结果校验断言
			if tt.expectContains != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectContains)) {
				t.Errorf("用例 [%s] 失败!\n预期包含: %s\n实际返回: %s", tt.name, tt.expectContains, actualBody)
			}
		})
	}
}
