package interceptor

import (
	"blog/internal/auth"
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type teamIDCtxKey struct{}

// 从context中取出调用方团队标识（JWT拦截器注入）
func TeamIDFromContext(ctx context.Context) (string, bool) {
	teamID, ok := ctx.Value(teamIDCtxKey{}).(string)
	return teamID, ok
}

//	gRPC Unary拦截器：统一校验二方JWT
//
// 从metadata的authorization字段取 "Bearer <token>"，校验通过才放行
func JwtAuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 1. 从metadata中取出authorization字段
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "缺少认证信息")
	}
	// 2. 校验authorization字段
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "缺少authorization请求头")
	}
	// 3. 校验authorization字段格式
	authHeader := authHeaders[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "authorization格式错误")
	}
	// 4. 取出token
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "token为空")
	}
	// 5. 校验token
	claims, err := auth.OpenVerifyToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	// 6. 校验通过，把调用方团队标识注入context
	ctx = context.WithValue(ctx, teamIDCtxKey{}, claims.TeamID)
	return handler(ctx, req)
}
