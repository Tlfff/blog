// Package grpc 提供带服务身份、Trace 与超时的内部 gRPC 客户端。
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	HeaderServiceID = "x-service-id"
	HeaderTraceID   = "x-trace-id"
)

// Dial 建立到内部服务的 gRPC 连接，并注入服务身份与 Trace metadata。
func Dial(addr, serviceID string, timeout time.Duration) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(ctx)
			md = metadata.Join(md, metadata.Pairs(
				HeaderServiceID, serviceID,
			))
			ctx = metadata.NewOutgoingContext(ctx, md)
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	)
}
