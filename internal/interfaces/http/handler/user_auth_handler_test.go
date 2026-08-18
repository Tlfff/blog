package handler

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	identityapp "blog/internal/application/identity"
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/user"
	identityinfra "blog/internal/infrastructure/identity"
	"blog/internal/model"
	"blog/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeSessionStore struct {
	values map[string]string
	sets   map[string]map[string]struct{}
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		values: make(map[string]string),
		sets:   make(map[string]map[string]struct{}),
	}
}

func (f *fakeSessionStore) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := f.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

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

func (f *fakeSessionStore) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	members := make([]string, 0, len(f.sets[key]))
	for member := range f.sets[key] {
		members = append(members, member)
	}
	return redis.NewStringSliceResult(members, nil)
}

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

func TestUserAuthHandler_AllRoutes(t *testing.T) {
	common.InitValidator()

	// 1. 核心修复：创建一个临时的纯内存 SQLite 数据库，用来给测试代码发泄数据
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法启动内存测试数据库: %v", err)
	}

	// 2.  自动迁移：让 GORM 默默在内存里把 users 表建出来
	_ = db.AutoMigrate(&model.User{})

	// 3.  完美对齐升级后的构造函数
	userRepo := repository.NewUserRepository(db)
	identityService := identityapp.NewService(
		identityinfra.NewUserRepository(userRepo),
		identityinfra.NewTokenSession(auth.NewTokenAuth(newFakeSessionStore())),
		nil,
		nil,
		"",
		nil,
	)
	userAuthService := identityService
	h := NewUserAuthHandler(userAuthService)

	// 4. 大表格：按“时光流逝”的顺序，先测异常，再测成功注册，最后测登录
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
			body: user.LoginRequest{
				Phone:    "18078789119", // 使用第2步成功注册的手机号
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
