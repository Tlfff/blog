// Package like 组装 Like 上下文的应用和基础设施依赖。
package like

import (
	likeapp "blog/internal/like/application"
	likedomain "blog/internal/like/domain"
	likeinfra "blog/internal/like/infrastructure"
	likehttp "blog/internal/like/interfaces/http"
	"blog/internal/platform/kafka"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 表示 Like 上下文对组合根公开的能力。
type Module struct {
	Application *likeapp.Service      // Like Application Facade
	HTTP        *likehttp.LikeHandler // Like HTTP Adapter
}

// NewModule 创建 Like 上下文模块。
func NewModule(
	db *gorm.DB,
	rdb *redis.Client,
	kafkaClient *kafka.Client,
	articleStatistics likedomain.ArticleLikeStatistics,
	commentStatistics likedomain.CommentLikeStatistics,
	tx likeapp.TransactionManager,
) *Module {
	articleRepo := likeinfra.NewArticleLikeRepository(db)
	commentRepo := likeinfra.NewCommentLikeRepository(db)
	cache := likeinfra.NewLikeCache(rdb, articleRepo, commentRepo)
	application := likeapp.NewService(
		articleRepo,
		commentRepo,
		cache,
		likeinfra.NewEventPublisher(kafkaClient),
		likeinfra.NewProjectionUpdater(articleStatistics, commentStatistics),
		tx,
	)
	return &Module{Application: application, HTTP: likehttp.NewLikeHandler(application, application)}
}
