// Package comment 组装 Comment 上下文的应用和基础设施依赖。
package comment

import (
	commentapp "blog/internal/comment/application"
	commentdomain "blog/internal/comment/domain"
	commentinfra "blog/internal/comment/infrastructure"
	commentgrpc "blog/internal/comment/interfaces/grpc"
	commenthttp "blog/internal/comment/interfaces/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Module 表示 Comment 上下文对组合根公开的能力。
type Module struct {
	Application    *commentapp.Service               // 评论 Application Facade
	LikeProjection *commentapp.LikeProjectionService // 评论点赞数 Application Facade
	HTTP           *commenthttp.CommentHandler       // Comment HTTP Adapter
	GRPC           *commentgrpc.CommentHandler       // Comment gRPC Adapter
}

// NewModule 创建 Comment 上下文模块。
func NewModule(db *gorm.DB, rdb *redis.Client, articleStatistics commentdomain.ArticleStatistics, tx commentapp.TransactionManager) *Module {
	repo := commentinfra.NewCommentRepository(db, articleStatistics)
	application := commentapp.NewService(repo, commentinfra.NewLikeCountQuery(rdb), tx)
	likeProjection := commentapp.NewLikeProjectionService(commentinfra.NewCommentLikeStatistics(db))
	return &Module{
		Application: application, LikeProjection: likeProjection,
		HTTP: commenthttp.NewCommentHandler(application),
		GRPC: commentgrpc.NewCommentHandler(application),
	}
}
