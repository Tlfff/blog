// Package grpc 提供带服务身份、Trace 与超时的内部 gRPC 客户端。
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// 内部gRPC metadata 字段名
const (
	HeaderServiceID = "x-service-id" // 调用方服务身份字段
	HeaderTraceID   = "x-trace-id"   // 链路追踪 Trace ID 字段
)

// 建立到内部服务的gRPC连接，并在每次调用时注入服务身份与超时控制
func Dial(addr, serviceID string, timeout time.Duration) (*grpc.ClientConn, error) {
	// 1. 创建 gRPC 客户端，内网调用使用非加密传输
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			// 2. 把调用方服务身份注入 outgoing metadata
			md, _ := metadata.FromOutgoingContext(ctx)
			md = metadata.Join(md, metadata.Pairs(
				HeaderServiceID, serviceID,
			))
			ctx = metadata.NewOutgoingContext(ctx, md)
			// 3. 配置了超时则为本次调用派生带超时的 context
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			// 4. 继续执行实际的 RPC 调用
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	)
}
