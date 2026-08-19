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

type fakeCommunityServer struct {
	internalv1.UnimplementedCommunityServiceServer
}

func (f *fakeCommunityServer) CreateComment(context.Context, *internalv1.CreateCommentRequest) (*internalv1.CreateCommentResponse, error) {
	return &internalv1.CreateCommentResponse{Id: 9, CreatedTime: 1700000000}, nil
}

func (f *fakeCommunityServer) GetHotRank(context.Context, *internalv1.HotRankRequest) (*internalv1.HotRank, error) {
	return &internalv1.HotRank{Items: []*internalv1.HotRankItem{{
		ArticleId: 1,
		Title:     "热榜文章",
		Hot:       18,
	}}}, nil
}

func (f *fakeCommunityServer) GetUnreadCount(context.Context, *internalv1.UserIDRequest) (*internalv1.UnreadCount, error) {
	return &internalv1.UnreadCount{Count: 3}, nil
}

func (f *fakeCommunityServer) GetCommentStats(context.Context, *internalv1.CommentIDRequest) (*internalv1.CommentStats, error) {
	return nil, status.Error(codes.NotFound, "评论不存在")
}

func newCommunityTestClient(t *testing.T) *CommunityClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	internalv1.RegisterCommunityServiceServer(s, &fakeCommunityServer{})
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
		t.Fatalf("创建 gRPC 连接失败: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return NewCommunityClient(internalv1.NewCommunityServiceClient(conn))
}

func TestCommunityClient_CoreMapping(t *testing.T) {
	client := newCommunityTestClient(t)
	ctx := context.Background()

	comment, err := client.CreateComment(ctx, 1, 0, 1, 0, "内容", "127.0.0.1")
	if err != nil {
		t.Fatalf("创建评论失败: %v", err)
	}
	if comment.ID != 9 || comment.CreatedTime != 1700000000 {
		t.Fatalf("创建评论映射不一致: %+v", comment)
	}

	rank, err := client.GetHotRank(ctx)
	if err != nil {
		t.Fatalf("获取热榜失败: %v", err)
	}
	if len(*rank.List) != 1 || (*rank.List)[0].Title != "热榜文章" || (*rank.List)[0].Hot != 18 {
		t.Fatalf("热榜映射不一致: %+v", rank.List)
	}

	count, err := client.GetUnreadCount(ctx, 1)
	if err != nil {
		t.Fatalf("获取未读数量失败: %v", err)
	}
	if count != 3 {
		t.Fatalf("未读数量映射不一致: %d", count)
	}

	if _, err := client.GetCommentStats(ctx, 1); err != common.ErrCommentNotFound {
		t.Fatalf("NotFound 应映射为 ErrCommentNotFound, got %v", err)
	}
}
