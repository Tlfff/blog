package handler

import (
	"blog/internal/service"
	"context"

	articlev1 "blog/gen/article"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ArticleHandler 实现二方服务的文章接口
type ArticleHandler struct {
	articlev1.UnimplementedArticleServiceServer
	articleService *service.ArticleService
}

func NewArticleHandler(articleService *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

// 获取全部可用（已发表）文章列表
func (h *ArticleHandler) GetPublishedList(ctx context.Context, req *articlev1.GetExternalListRequest) (*articlev1.ExternalListResponse, error) {
	// 1. 获取已发表文章列表
	resp, err := h.articleService.GetAvailabledList(ctx, req.Page, req.PageSize, req.IsDesc)
	if err != nil {
		return nil, status.Error(codes.Internal, "服务内部错误")
	}
	// 2. 构建返回响应
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
