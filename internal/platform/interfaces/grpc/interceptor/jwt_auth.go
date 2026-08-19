package interceptor

import (
	"blog/internal/platform/auth"
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// 认证拦截器（二方）：校验内部服务的JWT
type JwtInterceptor struct{}

// 构建二方JWT认证拦截器
func NewJwtInterceptor() *JwtInterceptor {
	return &JwtInterceptor{}
}

// 返回 gRPC 一元拦截器函数
func (j *JwtInterceptor) Unary() grpc.UnaryServerInterceptor {
	return j.Intercept
}

// 从metadata的authorization字段取 "Bearer <token>"，校验通过后把调用方身份注入context
func (j *JwtInterceptor) Intercept(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
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
	// 6. 校验通过，把调用方身份注入context（service_id 为授权主体，team_id 仅统计用）
	ctx = withIdentity(ctx, &Identity{
		Kind:  KindInternal,
		ID:    claims.ServiceID,
		Group: claims.TeamID,
	})
	return handler(ctx, req)
}
