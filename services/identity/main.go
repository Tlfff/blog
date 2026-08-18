// Identity Service 独立启动入口。
package main

import (
	identityapp "blog/internal/application/identity"
	"blog/internal/auth"
	"blog/internal/infrastructure/bootstrap"
	"blog/internal/infrastructure/config"
	identityinfra "blog/internal/infrastructure/identity"
	identitygrpc "blog/services/identity/internal/grpc"
	internalv1 "blog/shared/contracts/gen/internalv1"
	svcconfig "blog/shared/platform/config"
	"blog/shared/platform/server"
	"log"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := svcconfig.Load("services/identity/config.yaml")
	if err != nil {
		log.Fatalf("加载服务配置失败: %v", err)
	}
	business, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("加载业务配置失败: %v", err)
	}
	bootstrap.InitValidator()
	if err := bootstrap.InitIPSearcher(); err != nil {
		log.Fatalf("初始化 IP 解析器失败: %v", err)
	}
	defer bootstrap.CloseIP()

	db, err := bootstrap.NewMySQL(business)
	if err != nil {
		log.Fatalf("初始化 MySQL 失败: %v", err)
	}
	rdb, err := bootstrap.NewRedis(business)
	if err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}
	defer rdb.Close()
	ossClient, err := bootstrap.NewOSS(business)
	if err != nil {
		log.Fatalf("初始化 MinIO 失败: %v", err)
	}

	identityService := identityapp.NewService(
		identityinfra.NewUserRepository(db),
		identityinfra.NewTokenSession(auth.NewTokenAuth(rdb)),
		identityinfra.NewPasswordChangeTokenStore(rdb),
		identityinfra.NewAvatarStorage(ossClient),
		business.OSS.PublicDomain,
		business.OSS.AllowedExts,
	)

	allowed := allowedIDs(cfg)
	if err := server.Run(cfg.GRPCAddr, cfg.Name, allowed, func(s *grpc.Server) {
		internalv1.RegisterIdentityServiceServer(s, identitygrpc.NewIdentityServer(identityService))
	}); err != nil {
		log.Fatalf("Identity Service 启动失败: %v", err)
	}
}

func allowedIDs(cfg *svcconfig.Service) map[string]bool {
	allowed := map[string]bool{"gateway-service": true}
	for _, peer := range cfg.Peers {
		allowed[peer.Name+"-service"] = true
	}
	return allowed
}
