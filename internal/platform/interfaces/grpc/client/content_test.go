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

type fakeContentServer struct {
	internalv1.UnimplementedContentServiceServer
}

func (f *fakeContentServer) GetPublishedList(context.Context, *internalv1.ListArticlesRequest) (*internalv1.ArticleList, error) {
	return &internalv1.ArticleList{
		Items: []*internalv1.ArticleListItem{{
			Id:        1,
			Title:     "文章",
			AuthorId:  2,
			ViewCount: 3,
		}},
		Total:    1,
		LastId:   1,
		Page:     1,
		PageSize: 10,
	}, nil
}

func (f *fakeContentServer) GetAvailableList(context.Context, *internalv1.ListArticlesRequest) (*internalv1.AvailableList, error) {
	return &internalv1.AvailableList{
		Items: []*internalv1.AvailableItem{{
			Id:    1,
			Title: "开放文章",
			Tags:  []string{"Go"},
		}},
		Total:    1,
		Page:     1,
		PageSize: 10,
	}, nil
}

func (f *fakeContentServer) GetArticle(context.Context, *internalv1.ArticleUserRequest) (*internalv1.ArticleDetail, error) {
	return nil, status.Error(codes.NotFound, "文章不存在")
}

func newContentTestClient(t *testing.T) *ContentClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	internalv1.RegisterContentServiceServer(s, &fakeContentServer{})
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
	return NewContentClient(internalv1.NewContentServiceClient(conn))
}

func TestContentClient_ListsAndError(t *testing.T) {
	client := newContentTestClient(t)
	ctx := context.Background()

	list, err := client.GetPublishedList(ctx, 1, 10, 0, false)
	if err != nil {
		t.Fatalf("获取文章列表失败: %v", err)
	}
	if list.Total != 1 || len(list.List) != 1 || list.List[0].Title != "文章" {
		t.Fatalf("文章列表映射不一致: %+v", list)
	}

	external, err := client.GetAvailableList(ctx, 1, 10, false)
	if err != nil {
		t.Fatalf("获取开放列表失败: %v", err)
	}
	if external.Total != 1 || external.List[0].Tags[0] != "Go" {
		t.Fatalf("开放列表映射不一致: %+v", external)
	}

	if _, err := client.GetArticle(ctx, 1, 1); err != common.ErrArticleNotFound {
		t.Fatalf("NotFound 应映射为 ErrArticleNotFound, got %v", err)
	}
}
