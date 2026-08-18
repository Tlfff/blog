package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ServerAuthInterceptor 校验内部调用方的服务身份。
func ServerAuthInterceptor(allowedIDs map[string]bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if len(allowedIDs) == 0 {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "缺少服务身份")
		}
		ids := md.Get(HeaderServiceID)
		if len(ids) == 0 || !allowedIDs[ids[0]] {
			return nil, status.Error(codes.Unauthenticated, "未授权的服务调用")
		}
		return handler(ctx, req)
	}
}
