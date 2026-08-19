package client

import (
	"blog/internal/shared/common"
	internalv1 "blog/shared/contracts/gen/internalv1"
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fakeIdentityServer struct {
	internalv1.UnimplementedIdentityServiceServer
}

func (f *fakeIdentityServer) Register(context.Context, *internalv1.RegisterRequest) (*internalv1.Empty, error) {
	return nil, status.Error(codes.NotFound, "用户不存在")
}

func (f *fakeIdentityServer) Login(context.Context, *internalv1.LoginRequest) (*internalv1.LoginResponse, error) {
	return &internalv1.LoginResponse{AccessToken: "token-abc"}, nil
}

func (f *fakeIdentityServer) GetMyProfile(context.Context, *internalv1.UserIDRequest) (*internalv1.MyProfile, error) {
	return &internalv1.MyProfile{
		Id:            1,
		Nickname:      "用户",
		Avatar:        "avatar",
		Role:          1,
		LastLoginTime: 1700000000,
		LastLoginIp:   "内网",
	}, nil
}

func (f *fakeIdentityServer) GetUserBasicInfo(context.Context, *internalv1.UserIDRequest) (*internalv1.UserBasicInfo, error) {
	return &internalv1.UserBasicInfo{
		Id:            1,
		Nickname:      "用户",
		Avatar:        "avatar",
		LastLoginTime: 1700000000,
		LastLoginIp:   "内网",
	}, nil
}

func newIdentityTestClient(t *testing.T) *IdentityClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	internalv1.RegisterIdentityServiceServer(s, &fakeIdentityServer{})
	go func() {
		_ = s.Serve(lis)
	}()
	t.Cleanup(func() {
		s.Stop()
		_ = lis.Close()
	})

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("创建 gRPC 连接失败: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return NewIdentityClient(internalv1.NewIdentityServiceClient(conn))
}

func TestIdentityClient_LoginAndProfile(t *testing.T) {
	client := newIdentityTestClient(t)
	ctx := context.Background()

	resp, err := client.Login(ctx, "13800000000", "", "123456", "127.0.0.1", "web", false)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if resp.AccessToken != "token-abc" {
		t.Fatalf("登录响应字段不一致: %s", resp.AccessToken)
	}

	profile, err := client.GetMyProfile(ctx, 1)
	if err != nil {
		t.Fatalf("获取资料失败: %v", err)
	}
	if profile.ID != 1 || profile.Nickname != "用户" || profile.LastLoginIp != "内网" {
		t.Fatalf("资料映射不一致: %+v", profile)
	}

	info, err := client.GetUserBasicInfo(ctx, 1)
	if err != nil {
		t.Fatalf("获取用户基本信息失败: %v", err)
	}
	if info.ID != 1 || info.Nickname != "用户" {
		t.Fatalf("用户基本信息映射不一致: %+v", info)
	}
}

func TestIdentityClient_ErrorMapping(t *testing.T) {
	client := newIdentityTestClient(t)
	ctx := context.Background()

	if err := client.Register(ctx, "13800000000", "123456", "用户", "127.0.0.1"); err != common.ErrUserNotFound {
		t.Fatalf("NotFound 应映射为 ErrUserNotFound, got %v", err)
	}
}
