// Package article 组装 Article 上下文的应用和基础设施依赖。
package article

import (
	articleapp "blog/internal/article/application"
	articleinfra "blog/internal/article/infrastructure"
	articlegrpc "blog/internal/article/interfaces/grpc"
	articlehttp "blog/internal/article/interfaces/http"
	articlekafka "blog/internal/article/interfaces/kafka"
	platformconfig "blog/internal/platform/config"
	"blog/internal/platform/kafka"
	platformkafka "blog/internal/platform/kafka"
	platformoss "blog/internal/platform/oss"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// UserFacade 表示 Article 依赖的 User Application Facade。
type UserFacade = articleinfra.UserFacade

// LikeFacade 表示 Article 依赖的 Like Application Facade。
type LikeFacade = articleinfra.LikeFacade

// Module 表示 Article 上下文对组合根公开的能力。
type Module struct {
	Application *articleapp.Service           // 文章生命周期 Application Facade
	Engagement  *articleapp.EngagementService // 浏览历史和热榜 Application Facade
	Statistics  *articleapp.StatisticsService // 文章统计 Application Facade
	HTTP        *articlehttp.ArticleHandler   // Article HTTP Adapter
	GRPC        *articlegrpc.ArticleHandler   // Article gRPC Adapter
	Kafka       platformkafka.MessageHandler  // 浏览历史 Kafka Adapter
}

// NewModule 创建 Article 上下文模块。
func NewModule(
	db *gorm.DB,
	rdb *redis.Client,
	kafkaClient *kafka.Client,
	ossClient *platformoss.MinioClient,
	cfg *platformconfig.Config,
	users UserFacade,
	likes LikeFacade,
	statistics *articleapp.StatisticsService,
) *Module {
	repo := articleinfra.NewArticleRepository(db)
	if statistics == nil {
		statistics = articleapp.NewStatisticsService(articleinfra.NewArticleStatistics(db))
	}
	application := articleapp.NewService(
		repo,
		articleinfra.NewUserQuery(users),
		articleinfra.NewArticleImageStorage(ossClient),
		articleinfra.NewInteractionQuery(likes),
		cfg.OSS.PublicDomain,
		cfg.OSS.AllowedExts,
	)
	engagement := articleapp.NewEngagementService(
		articleinfra.NewViewHistoryRepository(db),
		articleinfra.NewHotRankStore(rdb),
		articleinfra.NewRankingQuery(db),
		articleinfra.NewViewEventPublisher(kafkaClient),
	)
	return &Module{
		Application: application, Engagement: engagement, Statistics: statistics,
		HTTP:  articlehttp.NewArticleHandler(application, engagement),
		GRPC:  articlegrpc.NewArticleHandler(application),
		Kafka: articlekafka.NewHandler(engagement),
	}
}
