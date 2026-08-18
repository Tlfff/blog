package handler

import (
	articledto "blog/internal/dto/article"
	"context"

	articlev1 "blog/gen/article"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ArticleQueryUsecase 是开放 gRPC 文章查询的应用用例接口。
type ArticleQueryUsecase interface {
	GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*articledto.ExternalListResponse, error)
}

// ArticleHandler 实现二方服务的文章接口
type ArticleHandler struct {
	articlev1.UnimplementedArticleServiceServer
	articleService ArticleQueryUsecase
}

func NewArticleHandler(articleService ArticleQueryUsecase) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

// 获取全部可用（已发表）文章列表
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
		return nil, GRPCError(err)
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
