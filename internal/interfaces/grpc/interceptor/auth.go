package interceptor

import (
	"blog/internal/infrastructure/config"
	"context"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// 统一认证拦截器：按 metadata 路由认证方式
// 有 x-access-key-id → 三方合作方，走 HMAC 校验
// 没有 → 内部服务，走 JWT 校验
type AuthInterceptor struct {
	jwt  *JwtInterceptor
	hmac *HmacAuthInterceptor
}

// 构建统一认证拦截器
func NewAuthInterceptor(rdb *redis.Client, partners []config.Partner) *AuthInterceptor {
	return &AuthInterceptor{
		jwt:  NewJwtInterceptor(),
		hmac: NewHmacAuthInterceptor(rdb, partners),
	}
}

// 返回 gRPC 一元拦截器函数
func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return a.Intercept
}

// 统一认证入口，只回答"这次调用的身份是谁"
func (a *AuthInterceptor) Intercept(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 判断调用方身份类型：外部合作方会携带 access_key_id
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md.Get(mdKeyAccessKeyID)) > 0 {
			// 三方：HMAC 签名校验
			return a.hmac.Intercept(ctx, req, info, handler)
		}
	}
	// 二方：JWT 校验
	return a.jwt.Intercept(ctx, req, info, handler)
}
