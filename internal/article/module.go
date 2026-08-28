// Package article 组装 Article 上下文的应用和基础设施依赖。
package article

import (
	articleapp "blog/internal/article/app"
	articleinfra "blog/internal/article/infra"
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
//
// 参数说明：
//   - db：MySQL 数据库连接。
//   - rdb：Redis 客户端。
//   - kafkaClient：Kafka 客户端。
//   - ossClient：MinIO 对象存储客户端。
//   - cfg：应用配置。
//   - users：User Application Facade。
//   - likes：Like Application Facade。
//   - statistics：文章统计服务，可以为空。
//   - tx：本地数据库事务协调器。
func NewModule(
	db *gorm.DB,
	rdb *redis.Client,
	kafkaClient *kafka.Client,
	ossClient *platformoss.MinioClient,
	cfg *platformconfig.Config,
	users UserFacade,
	likes LikeFacade,
	statistics *articleapp.StatisticsService,
	tx articleapp.TransactionManager,
) *Module {
	repo := articleinfra.NewArticleRepository(db)
	if statistics == nil {
		statistics = articleapp.NewStatisticsService(articleinfra.NewArticleStatistics(db))
	}
	application := articleapp.NewService(articleapp.ServiceDependencies{
		Articles:        repo,
		ArticleImages:   articleinfra.NewArticleImageRepository(db),
		ImageReferences: articleinfra.NewMarkdownImageReferenceParser(),
		Users:           articleinfra.NewUserQuery(users),
		ImageStorage:    articleinfra.NewArticleImageStorage(ossClient),
		Interactions:    articleinfra.NewInteractionQuery(likes),
		Transactions:    tx,
		PublicDomain:    cfg.OSS.PublicDomain,
		AllowedExts:     cfg.OSS.AllowedExts,
	})
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
