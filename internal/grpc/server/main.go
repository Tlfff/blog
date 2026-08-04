package server

import (
	"blog/config"
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

// 创建gRPC Server，注册所有服务，挂载认证（Auth）+ 授权（AuthZ）拦截器链
func NewGRPCServer(appHandler *AppHandler, partners []config.Partner) *grpc.Server {
	// 1. 创建 gRPC Server，挂统一认证拦截器（二方JWT / 三方HMAC）+ 授权拦截器
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.NewAuthInterceptor(partners).Unary(), // 认证：回答"身份是谁"
			interceptor.NewAuthzInterceptor(nil).Unary(),     // 授权：回答"能否调此方法"（权限表暂未实现，先放行）
		),
	)
	// 2. 注册服务
	userv1.RegisterUserServiceServer(s, appHandler.User)
	commentv1.RegisterCommentServiceServer(s, appHandler.Comment)
	articlev1.RegisterArticleServiceServer(s, appHandler.Article)
	return s
}
