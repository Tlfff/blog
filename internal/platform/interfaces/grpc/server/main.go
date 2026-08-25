package server

import (
	articlev1 "blog/gen/article"
	commentv1 "blog/gen/comment"
	userv1 "blog/gen/user"
	articlegrpc "blog/internal/article/interfaces/grpc"
	commentgrpc "blog/internal/comment/interfaces/grpc"
	"blog/internal/platform/config"
	"blog/internal/platform/interfaces/grpc/interceptor"
	usergrpc "blog/internal/user/interfaces/grpc"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

// AppHandler 汇总对外 gRPC Service Adapter。
type AppHandler struct {
	Article *articlegrpc.ArticleHandler // Article gRPC Adapter
	User    *usergrpc.UserHandler       // User gRPC Adapter
	Comment *commentgrpc.CommentHandler // Comment gRPC Adapter
}

// NewGRPCServer 创建并注册对外 gRPC 服务。
func NewGRPCServer(appHandler *AppHandler, rdb *redis.Client, partners []config.Partner) *grpc.Server {
	// 1. 按日志、认证顺序创建一元拦截器链
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		interceptor.LoggingInterceptor,
		interceptor.NewAuthInterceptor(rdb, partners).Unary(),
	))

	// 2. 注册现有 User、Comment 和 Article Service
	userv1.RegisterUserServiceServer(server, appHandler.User)
	commentv1.RegisterCommentServiceServer(server, appHandler.Comment)
	articlev1.RegisterArticleServiceServer(server, appHandler.Article)
	return server
}
