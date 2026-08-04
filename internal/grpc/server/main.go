package server

import (
	articlev1 "blog/gen/article"
	commentv1 "blog/gen/comment"
	userv1 "blog/gen/user"
	grpchandler "blog/internal/grpc/handler"
	"blog/internal/grpc/interceptor"

	"google.golang.org/grpc"
)

// 未来增加新模块，只需在这里加一行
type AppHandler struct {
	Article *grpchandler.ArticleHandler
	User    *grpchandler.UserHandler
	Comment *grpchandler.CommentHandler
}

// 创建gRPC Server，注册所有二方服务和JWT拦截器
func NewGRPCServer(appHandler *AppHandler) *grpc.Server {
	// 1. 创建 gRPC Server
	s := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.JwtAuthInterceptor), // 注册 JWT拦截器,一元拦截器
	)
	// 2. 注册服务
	userv1.RegisterUserServiceServer(s, appHandler.User)
	commentv1.RegisterCommentServiceServer(s, appHandler.Comment)
	articlev1.RegisterArticleServiceServer(s, appHandler.Article)
	return s
}
