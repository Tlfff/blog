package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthorizeFunc 授权判定函数：返回 nil 表示允许，否则返回拒绝原因。
// 目前方法权限表暂未实现，先传 nil（所有已认证请求放行）；
// 后续实现权限表时，通过该函数注入"方法 -> 允许身份列表"的映射判断。
type AuthorizeFunc func(ctx context.Context, identity *Identity, fullMethod string) error

// AuthzInterceptor 授权拦截器：只回答"这个身份能不能调这个方法"。
// 认证拦截器在前，授权拦截器在后，两层的职责完全隔离。
type AuthzInterceptor struct {
	authorize AuthorizeFunc
}

// NewAuthzInterceptor 构建授权拦截器
func NewAuthzInterceptor(authorize AuthorizeFunc) *AuthzInterceptor {
	return &AuthzInterceptor{authorize: authorize}
}

// Unary 返回 gRPC 一元拦截器函数
func (a *AuthzInterceptor) Unary() grpc.UnaryServerInterceptor {
	return a.Intercept
}

// Intercept 授权判定：先确认请求已认证，再执行授权函数
func (a *AuthzInterceptor) Intercept(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 1. 从 context 取出认证拦截器注入的身份；取不到说明认证层没跑或未通过
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "请求未认证")
	}
	// 2. 执行授权判定（未配置权限表时放行所有已认证请求）
	if a.authorize != nil {
		if err := a.authorize(ctx, identity, info.FullMethod); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}
	return handler(ctx, req)
}
