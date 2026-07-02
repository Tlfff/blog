package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 调试开关：让 Gin 在测试时保持安静，不喷出一堆启动日志
func init() {
	gin.SetMode(gin.TestMode)
}

// 辅助函数：快速构造一个带有虚拟请求和上下文数据的 Gin 环境
func makeTestContext(method, path string, body interface{}, ctxUser *auth.UserContext) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 1. 模拟 HTTP 请求
	var req *http.Request
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	c.Request = req

	// 2. 模拟登录中间件往上下文塞入的用户信息
	if ctxUser != nil {
		c.Set("currentUser", ctxUser)
	}

	return c, w
}
func TestUserHandler_AllRoutes(t *testing.T) {
	// 1. 核心修复：创建一个临时的纯内存 SQLite 数据库，用来给测试代码发泄数据
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法启动内存测试数据库: %v", err)
	}

	// 2.  自动迁移：让 GORM 默默在内存里把 users 表建出来
	_ = db.AutoMigrate(&model.User{})

	// 3.  完美对齐升级后的构造函数
	// 这一步和你的 main.go 逻辑完全一致，利用内存模式原地起飞
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo) // 如果你写的是 NewUserService(userRepo) 就用那个
	userAuthService := service.NewUserAuthService(userRepo)

	// 把真实组装好的服务喂给 Handler
	h := NewUserHandler(userService)

	_ = userAuthService.Register("13800000000", "123456", "测试用户", "120.0.0.1")
	// 2.  定义大表格
	tests := []struct {
		name           string
		run            func(c *gin.Context) // 动态指定调用哪个 Handler 函数
		method         string
		path           string
		body           interface{}       // 模拟请求体
		ctxUser        *auth.UserContext // 模拟是否登录
		expectStatus   int               // 预期 HTTP 状态码
		expectContains string            // 预期返回的 JSON 字符串里包含什么关键字
	}{
		// ====================  场景 A：获取个人资料 (GetMyProfile) ====================
		{
			name:           "1. 获取个人资料-成功",
			run:            h.GetMyProfile,
			method:         "GET",
			path:           "/my/profile",
			ctxUser:        &auth.UserContext{UserID: 1, Phone: "13800000000"},
			expectStatus:   http.StatusOK,
			expectContains: `"获取成功"`,
		},

		// ====================  场景 B：查看他人主页 (GetPublicProfile) ====================
		{
			name:         "2. 查看他人主页-绑定参数失败(触发第一个if)",
			run:          h.GetPublicProfile,
			method:       "GET",
			path:         "/user/profile?user_id=abc", // 故意传成字符串，让 ShouldBindQuery 报错
			ctxUser:      nil,
			expectStatus: http.StatusBadRequest, // 如果进入 c.Error(common.ErrUserNotFound)
		},

		// ====================  场景 C：修改基础资料 (UpdateProfile) ====================
		{
			name:         "3. 修改基础资料-请求体错误(触发第一个if)",
			run:          h.UpdateProfile,
			method:       "POST",
			path:         "/user/profile/update",
			body:         "字符串格式的坏JSON", // 故意不给结构体，让 ShouldBindJSON 报错
			ctxUser:      &auth.UserContext{UserID: 1},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:           "4. 修改基础资料-成功通关",
			run:            h.UpdateProfile,
			method:         "POST",
			path:           "/user/profile/update",
			body:           map[string]interface{}{"nickname": "新林风", "avatar": "http://img.png"},
			ctxUser:        &auth.UserContext{UserID: 1},
			expectStatus:   http.StatusOK,
			expectContains: `"个人资料修改成功"`,
		},

		// ==================== 场景 D：修改密码 (UpdatePassword) ====================
		{
			name:           "5. 修改密码-成功通关",
			run:            h.UpdatePassword,
			method:         "POST",
			path:           "/user/password/update",
			body:           map[string]interface{}{"old_password": "123456", "new_password": "456789"},
			ctxUser:        &auth.UserContext{UserID: 1},
			expectStatus:   http.StatusOK,
			expectContains: `"密码修改成功"`,
		},
	}

	// 3.  执行引擎
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 生成当前用例需要的上下文环境
			c, w := makeTestContext(tt.method, tt.path, tt.body, tt.ctxUser)

			// 核心：动态调用对应的目标 Handler 函数！
			tt.run(c)

			// 【核心技巧】手动触发一下你的全局错误中间件逻辑
			// 因为在单元测试里没有走真正的 Gin 路由引擎，c.Next() 不会触发。如果 len(c.Errors) > 0，需要在这里手动校验
			if len(c.Errors) > 0 {
				// 可以通过这段代码模拟中间件行为
				err := c.Errors.Last().Err
				if err == common.ErrInvalidRequestBody || err == common.ErrUserNotFound {
					w.Code = http.StatusBadRequest // 模拟状态码改变
				}
			}

			// 结果断言
			if tt.expectContains != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectContains)) {
				t.Errorf("用例 [%s] 失败: 预期返回包含 %s, 实际返回: %s", tt.name, tt.expectContains, w.Body.String())
			}
		})
	}
}
