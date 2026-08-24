package grpc

import (
	articledto "blog/internal/article/app/dto"
	grpcinterface "blog/internal/platform/interfaces/grpc"
	"context"

	articlev1 "blog/gen/article"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ArticleQueryUsecase 是开放 gRPC 文章查询的应用用例接口。
type ArticleQueryUsecase interface {
	// GetAvailableList 查询对外开放的已发表文章列表。
	GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*articledto.ExternalListResponse, error)
}

// ArticleHandler 实现二方服务的文章接口
type ArticleHandler struct {
	articlev1.UnimplementedArticleServiceServer                     // 保持向前兼容的未实现方法
	articleService                              ArticleQueryUsecase // 文章查询应用用例
}

// NewArticleHandler 创建 Article gRPC Handler。
func NewArticleHandler(articleService ArticleQueryUsecase) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

// GetAvailableList 获取全部可用的已发表文章列表。
func (h *ArticleHandler) GetAvailableList(ctx context.Context, req *articlev1.GetExternalListRequest) (*articlev1.ExternalListResponse, error) {
	// 1. 入参校验
	if req.Page <= 0 {
		return nil, status.Error(codes.InvalidArgument, "page必须大于0")
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page_size必须在1到100之间")
	}
	// 2. 获取已发表文章列表
	resp, err := h.articleService.GetAvailableList(ctx, req.Page, req.PageSize, req.IsDesc)
	if err != nil {
		return nil, grpcinterface.Error(err)
	}
	// 3. 构建返回响应
	items := make([]*articlev1.ArticleItem, 0, len(resp.List))
	for _, item := range resp.List {
		items = append(items, &articlev1.ArticleItem{
			Id:          item.ID,
			Title:       item.Title,
			Tags:        item.Tags,
			CreatedTime: item.CreatedTime,
			UpdatedTime: item.UpdatedTime,
		})
	}
	return &articlev1.ExternalListResponse{
		Items:    items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}
