package server

import (
	"blog/internal/infrastructure/config"
	articlev1 "blog/gen/article"
	commentv1 "blog/gen/comment"
	userv1 "blog/gen/user"
	grpchandler "blog/internal/interfaces/grpc/handler"
	"blog/internal/interfaces/grpc/interceptor"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

// 未来增加新模块，只需在这里加一行
type AppHandler struct {
	Article *grpchandler.ArticleHandler
	User    *grpchandler.UserHandler
	Comment *grpchandler.CommentHandler
}

// 创建gRPC Server，注册所有服务，挂载认证拦截器链
func NewGRPCServer(appHandler *AppHandler, rdb *redis.Client, partners []config.Partner) *grpc.Server {
	// 1. 创建 gRPC Server，拦截器链：日志 → 认证（二方JWT / 三方HMAC）
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.LoggingInterceptor,                        // 日志：进入/退出全量记录
			interceptor.NewAuthInterceptor(rdb, partners).Unary(), // 认证：不同调用，走不同认证方式
		),
	)
	// 2. 注册服务
	userv1.RegisterUserServiceServer(s, appHandler.User)
	commentv1.RegisterCommentServiceServer(s, appHandler.Comment)
	articlev1.RegisterArticleServiceServer(s, appHandler.Article)
	return s
}
