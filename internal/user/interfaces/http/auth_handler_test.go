package http

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"blog/internal/platform/interfaces/http/validation"
	"blog/internal/platform/security"
	identityapp "blog/internal/user/app"
	identityinfra "blog/internal/user/infra"
	"blog/internal/user/infra/model"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// fakeSessionStore 是 User HTTP 测试使用的内存 Redis 会话存储。
type fakeSessionStore struct {
	values map[string]string              // Redis 字符串键值映射
	sets   map[string]map[string]struct{} // Redis Set 键值映射
}

// newFakeSessionStore 创建内存会话存储。
func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		values: make(map[string]string),
		sets:   make(map[string]map[string]struct{}),
	}
}

// Get 查询字符串值。
func (f *fakeSessionStore) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := f.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

// Set 保存字符串值。
func (f *fakeSessionStore) Set(_ context.Context, key string, value any, _ time.Duration) *redis.StatusCmd {
	switch v := value.(type) {
	case []byte:
		f.values[key] = string(v)
	case string:
		f.values[key] = v
	default:
		f.values[key] = fmt.Sprint(v)
	}
	return redis.NewStatusResult("OK", nil)
}

// Del 删除字符串或 Set 键。
func (f *fakeSessionStore) Del(_ context.Context, keys ...string) *redis.IntCmd {
	deleted := int64(0)
	for _, key := range keys {
		if _, ok := f.values[key]; ok {
			delete(f.values, key)
			deleted++
		}
		delete(f.sets, key)
	}
	return redis.NewIntResult(deleted, nil)
}

// SAdd 向 Set 中添加成员。
func (f *fakeSessionStore) SAdd(_ context.Context, key string, members ...any) *redis.IntCmd {
	if f.sets[key] == nil {
		f.sets[key] = make(map[string]struct{})
	}
	added := int64(0)
	for _, member := range members {
		m := fmt.Sprint(member)
		if _, exists := f.sets[key][m]; !exists {
			f.sets[key][m] = struct{}{}
			added++
		}
	}
	return redis.NewIntResult(added, nil)
}

// SMembers 查询 Set 全部成员。
func (f *fakeSessionStore) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	members := make([]string, 0, len(f.sets[key]))
	for member := range f.sets[key] {
		members = append(members, member)
	}
	return redis.NewStringSliceResult(members, nil)
}

// SRem 从 Set 中删除成员。
func (f *fakeSessionStore) SRem(_ context.Context, key string, members ...any) *redis.IntCmd {
	removed := int64(0)
	for _, member := range members {
		m := fmt.Sprint(member)
		if _, exists := f.sets[key][m]; exists {
			delete(f.sets[key], m)
			removed++
		}
	}
	return redis.NewIntResult(removed, nil)
}

// TestUserAuthHandler_AllRoutes 验证注册和登录 Handler 的主要场景。
func TestUserAuthHandler_AllRoutes(t *testing.T) {
	validation.InitValidator()

	// 1. 创建临时内存 SQLite 数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法启动内存测试数据库: %v", err)
	}

	// 2. 创建 users 测试表
	_ = db.AutoMigrate(&model.User{})

	// 3. 按生产依赖关系组装 User Application 和 Handler
	identityService := identityapp.NewService(
		identityinfra.NewUserRepository(db),
		identityinfra.NewTokenSession(auth.NewTokenAuth(newFakeSessionStore())),
		nil,
		nil,
		identityinfra.NewPasswordHasher(),
		"",
		nil,
	)
	userAuthService := identityService
	h := NewUserAuthHandler(userAuthService)

	// 4. 定义注册和登录测试场景
	tests := []struct {
		name           string               // 测试场景名称
		run            func(c *gin.Context) // 动态调用目标函数
		method         string               // HTTP Method
		path           string               // 请求路径
		body           interface{}          // 请求体
		ctxUser        *auth.UserContext    // 模拟登录用户，可以为空
		expectContains string               // 预期返回包含的内容
	}{
		// 注册场景
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
			body: RegisterRequest{
				Nickname: "林风",
				Phone:    "18078789119",
				Password: "123456",
			},
			ctxUser:        nil,
			expectContains: `"注册成功"`,
		},

		// 登录场景
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
			body: LoginRequest{
				Phone:    "18078789119",
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
			body: LoginRequest{
				Phone:    "18078789119", // 使用第2步成功注册的手机号
				Password: "123456",
			},
			ctxUser:        nil,
			expectContains: `"access_token"`, // 成功登录应该能拿到你的令牌字段
		},
	}

	// 5. 逐场景执行 Handler 并校验响应
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 5.1 构造 Gin 测试上下文
			c, w := makeTestContext(tt.method, tt.path, tt.body, tt.ctxUser)

			// 5.2 执行目标 Handler
			tt.run(c)

			// 5.3 收集响应或 Gin Error，便于失败定位
			actualBody := w.Body.String()
			if actualBody == "" && len(c.Errors) > 0 {
				actualBody = "[被 c.Error 拦截] 原因: " + c.Errors.Last().Error()
			}

			// 5.4 校验响应关键字段
			if tt.expectContains != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectContains)) {
				t.Errorf("用例 [%s] 失败!\n预期包含: %s\n实际返回: %s", tt.name, tt.expectContains, actualBody)
			}
		})
	}
}
