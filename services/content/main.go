// Content Service 独立启动入口。
package main

import (
	contentapp "blog/internal/article/application"
	contentinfra "blog/internal/article/infrastructure"
	"blog/internal/platform/bootstrap"
	"blog/internal/platform/config"
	grpcclient "blog/internal/platform/interfaces/grpc/client"
	contentgrpc "blog/services/content/internal/grpc"
	internalv1 "blog/shared/contracts/gen/internalv1"
	svcconfig "blog/shared/platform/config"
	platformgrpc "blog/shared/platform/grpc"
	"blog/shared/platform/server"
	"log"
	"time"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := svcconfig.Load("services/content/config.yaml")
	if err != nil {
		log.Fatalf("加载服务配置失败: %v", err)
	}
	business, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("加载业务配置失败: %v", err)
	}

	db, err := bootstrap.NewMySQL(business)
	if err != nil {
		log.Fatalf("初始化 MySQL 失败: %v", err)
	}
	ossClient, err := bootstrap.NewOSS(business)
	if err != nil {
		log.Fatalf("初始化 MinIO 失败: %v", err)
	}

	identityConn, err := platformgrpc.Dial(cfg.PeerAddr("identity"), "content-service", 3*time.Second)
	if err != nil {
		log.Fatalf("连接 Identity Service 失败: %v", err)
	}
	defer identityConn.Close()
	communityConn, err := platformgrpc.Dial(cfg.PeerAddr("community"), "content-service", 3*time.Second)
	if err != nil {
		log.Fatalf("连接 Community Service 失败: %v", err)
	}
	defer communityConn.Close()

	contentService := contentapp.NewService(
		contentinfra.NewArticleRepository(db),
		grpcclient.NewContentUserQueryClient(internalv1.NewIdentityServiceClient(identityConn)),
		contentinfra.NewArticleImageStorage(ossClient),
		grpcclient.NewContentInteractionClient(internalv1.NewCommunityServiceClient(communityConn)),
		business.OSS.PublicDomain,
		business.OSS.AllowedExts,
	)

	allowed := map[string]bool{"gateway-service": true}
	for _, peer := range cfg.Peers {
		allowed[peer.Name+"-service"] = true
	}
	if err := server.Run(cfg.GRPCAddr, cfg.Name, allowed, func(s *grpc.Server) {
		internalv1.RegisterContentServiceServer(s, contentgrpc.NewContentServer(contentService))
	}); err != nil {
		log.Fatalf("Content Service 启动失败: %v", err)
	}
}
