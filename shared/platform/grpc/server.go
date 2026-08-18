package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// 构造服务端拦截器，校验内部调用方的服务身份
func ServerAuthInterceptor(allowedIDs map[string]bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// 1. 未配置白名单表示不校验，直接放行
		if len(allowedIDs) == 0 {
			return handler(ctx, req)
		}
		// 2. 取出请求 metadata，缺失则视为未认证
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "缺少服务身份")
		}
		// 3. 校验 service id 是否在白名单内
		ids := md.Get(HeaderServiceID)
		if len(ids) == 0 || !allowedIDs[ids[0]] {
			return nil, status.Error(codes.Unauthenticated, "未授权的服务调用")
		}
		// 4. 校验通过，继续执行业务处理
		return handler(ctx, req)
	}
}
