// Package user 组装 User 上下文的应用和基础设施依赖。
package user

import (
	platformconfig "blog/internal/platform/config"
	platformoss "blog/internal/platform/oss"
	platformsecurity "blog/internal/platform/security"
	userapp "blog/internal/user/application"
	userinfra "blog/internal/user/infrastructure"
	usergrpc "blog/internal/user/interfaces/grpc"
	userhttp "blog/internal/user/interfaces/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 表示 User 上下文对组合根公开的能力。
type Module struct {
	Application *userapp.Service          // User Application Facade
	HTTPAuth    *userhttp.UserAuthHandler // 用户注册登录 HTTP Adapter
	HTTP        *userhttp.UserHandler     // 用户资料 HTTP Adapter
	GRPC        *usergrpc.UserHandler     // User gRPC Adapter
}

// NewModule 创建 User 上下文模块。
func NewModule(db *gorm.DB, rdb *redis.Client, ossClient *platformoss.MinioClient, cfg *platformconfig.Config) *Module {
	repo := userinfra.NewUserRepository(db)
	sessions := userinfra.NewTokenSession(platformsecurity.NewTokenAuth(rdb))
	passwordTokens := userinfra.NewPasswordChangeTokenStore(rdb)
	avatars := userinfra.NewAvatarStorage(ossClient)
	passwords := userinfra.NewPasswordHasher()
	application := userapp.NewService(
		repo, sessions, passwordTokens, avatars, passwords,
		cfg.OSS.PublicDomain, cfg.OSS.AllowedExts,
	)
	return &Module{
		Application: application,
		HTTPAuth:    userhttp.NewUserAuthHandler(application),
		HTTP:        userhttp.NewUserHandler(application, application),
		GRPC:        usergrpc.NewUserHandler(application),
	}
}
