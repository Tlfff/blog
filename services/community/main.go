// Community Service 独立启动入口，承载互动、统计、通知与 Kafka 消费者。
package main

import (
	articleapp "blog/internal/article/application"
	articleinfra "blog/internal/article/infrastructure"
	commentapp "blog/internal/comment/application"
	commentinfra "blog/internal/comment/infrastructure"
	likeapp "blog/internal/like/application"
	likeinfra "blog/internal/like/infrastructure"
	notificationapp "blog/internal/notification/application"
	notificationinfra "blog/internal/notification/infrastructure"
	"blog/internal/platform/bootstrap"
	"blog/internal/platform/config"
	mq "blog/internal/platform/interfaces/mq"
	communitygrpc "blog/services/community/internal/grpc"
	internalv1 "blog/shared/contracts/gen/internalv1"
	svcconfig "blog/shared/platform/config"
	platformgrpc "blog/shared/platform/grpc"
	"blog/shared/platform/server"
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := svcconfig.Load("services/community/config.yaml")
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
	mongodb, err := bootstrap.NewMongoDB(business)
	if err != nil {
		log.Fatalf("初始化 MongoDB 失败: %v", err)
	}
	rdb, err := bootstrap.NewRedis(business)
	if err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}
	defer rdb.Close()
	if group := cfg.KafkaGroups.Notification; group != "" {
		if topic, ok := business.Kafka.Topics["notification"]; ok {
			topic.GroupID = group
			business.Kafka.Topics["notification"] = topic
		}
	}
	if group := cfg.KafkaGroups.ViewHistory; group != "" {
		if topic, ok := business.Kafka.Topics["view_history"]; ok {
			topic.GroupID = group
			business.Kafka.Topics["view_history"] = topic
		}
	}
	kafkaClient, err := bootstrap.NewKafka(business)
	if err != nil {
		log.Fatalf("初始化 Kafka 失败: %v", err)
	}
	defer kafkaClient.Close()

	identityConn, err := platformgrpc.Dial(cfg.PeerAddr("identity"), "community-service", 3*time.Second)
	if err != nil {
		log.Fatalf("连接 Identity Service 失败: %v", err)
	}
	defer identityConn.Close()
	contentConn, err := platformgrpc.Dial(cfg.PeerAddr("content"), "community-service", 3*time.Second)
	if err != nil {
		log.Fatalf("连接 Content Service 失败: %v", err)
	}
	defer contentConn.Close()

	articleLikePort := likeinfra.NewArticleLikeRepository(db, articleinfra.NewArticleLikeStatistics(db))
	commentLikePort := likeinfra.NewCommentLikeRepository(db, commentinfra.NewCommentLikeStatistics(db))
	commentRepo := commentinfra.NewCommentRepository(db, articleinfra.NewArticleStatistics(db))
	contentClient := internalv1.NewContentServiceClient(contentConn)
	identityClient := internalv1.NewIdentityServiceClient(identityConn)
	commentService := commentapp.NewService(commentRepo, commentinfra.NewArticleQuery(contentClient), likeinfra.NewLikeCountStore(rdb), commentinfra.NewCommentEventPublisher(kafkaClient))
	likeService := likeapp.NewService(articleLikePort, commentLikePort, likeinfra.NewTargetQuery(contentClient, commentRepo), likeinfra.NewEventPublisher(kafkaClient))
	articleService := articleapp.NewEngagementService(articleinfra.NewViewHistoryRepository(db), articleinfra.NewHotRankStore(rdb), articleinfra.NewRankingQuery(db), articleinfra.NewViewEventPublisher(kafkaClient))
	notificationService := notificationapp.NewService(notificationinfra.NewNotificationRepository(mongodb), notificationinfra.NewArticleQuery(contentClient), notificationinfra.NewUserInfoQuery(identityClient), notificationinfra.NewCommentQuery(commentRepo))

	if err := articleService.RebuildHotRank(context.Background()); err != nil {
		log.Printf("[WARN] Community Service 热榜初始化失败: %v", err)
	}

	// Kafka 消费者由 Community Service 承载。
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()
	if err := kafkaClient.InitConsumer(mq.RegisterHandlers(notificationService, articleService)); err != nil {
		log.Fatalf("初始化 Kafka 消费者失败: %v", err)
	}
	go func() {
		if err := kafkaClient.StartConsumer(consumerCtx); err != nil {
			log.Printf("[WARN] Kafka 消费者运行异常: %v", err)
		}
	}()

	allowed := map[string]bool{"gateway-service": true}
	for _, peer := range cfg.Peers {
		allowed[peer.Name+"-service"] = true
	}
	if err := server.Run(cfg.GRPCAddr, cfg.Name, allowed, func(s *grpc.Server) {
		internalv1.RegisterCommunityServiceServer(s, communitygrpc.NewCommunityServer(commentService, likeService, articleService, notificationService))
	}); err != nil {
		log.Fatalf("Community Service 启动失败: %v", err)
	}
}
