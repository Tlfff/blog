package grpc

import (
	"context"
	"net"
	"testing"

	internalv1 "blog/shared/contracts/gen/internalv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type testPingServer struct {
	grpc.ServiceRegistrar
}

type fakeIdentityServer struct {
	internalv1.UnimplementedIdentityServiceServer
}

func (f *fakeIdentityServer) GetUserBasicInfo(_ context.Context, _ *internalv1.UserIDRequest) (*internalv1.UserBasicInfo, error) {
	return &internalv1.UserBasicInfo{Id: 1, Nickname: "用户"}, nil
}

func TestServerAuthInterceptor(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	allowed := map[string]bool{"identity-service": true, "gateway-service": true}
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(ServerAuthInterceptor(allowed)))
	internalv1.RegisterIdentityServiceServer(s, &fakeIdentityServer{})
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(func() {
		s.Stop()
		_ = lis.Close()
	})

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("创建连接失败: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// 未携带服务身份应被拒绝。
	err = conn.Invoke(context.Background(), "/blog.internal.v1.IdentityService/GetUserBasicInfo", &internalv1.UserIDRequest{}, &internalv1.UserBasicInfo{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("未授权调用应返回 Unauthenticated, got %v", err)
	}

	// 允许列表内的服务身份应通过（业务方法未注册，返回 Unimplemented 而不是 Unauthenticated）。
	ctx := metadata.AppendToOutgoingContext(context.Background(), HeaderServiceID, "identity-service")
	var resp internalv1.UserBasicInfo
	err = conn.Invoke(ctx, "/blog.internal.v1.IdentityService/GetUserBasicInfo", &internalv1.UserIDRequest{}, &resp)
	if err != nil {
		t.Fatalf("已授权调用应成功: %v", err)
	}
	if resp.Id != 1 || resp.Nickname != "用户" {
		t.Fatalf("响应映射不一致: id=%d nickname=%s", resp.Id, resp.Nickname)
	}
}
