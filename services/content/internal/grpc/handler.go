// Package grpc 实现 Content Service 的内部 gRPC 接口。
package grpc

import (
	contentapp "blog/internal/application/content"
	articledto "blog/internal/dto/article"
	internalv1 "blog/shared/contracts/gen/internalv1"
	platformerrors "blog/shared/platform/errors"
	"context"
	"strings"
)

// ContentServer 把 Content Application 用例暴露为内部 gRPC 服务。
type ContentServer struct {
	internalv1.UnimplementedContentServiceServer
	app *contentapp.Service
}

func NewContentServer(app *contentapp.Service) *ContentServer {
	return &ContentServer{app: app}
}

func (s *ContentServer) CreateArticle(ctx context.Context, req *internalv1.CreateArticleRequest) (*internalv1.Empty, error) {
	if err := s.app.CreateArticle(ctx, req.AuthorId, req.Title, req.Content, req.Tags, int8(req.Status)); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *ContentServer) UpdateArticle(ctx context.Context, req *internalv1.UpdateArticleRequest) (*internalv1.Empty, error) {
	if err := s.app.UpdateArticle(ctx, req.ArticleId, req.AuthorId, req.Title, req.Content, req.Tags, int8(req.Status)); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *ContentServer) DeleteArticle(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.Empty, error) {
	if err := s.app.DeleteArticle(ctx, req.ArticleId, req.UserId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *ContentServer) ClearArticle(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.Empty, error) {
	if err := s.app.ClearArticle(ctx, req.ArticleId, req.UserId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *ContentServer) PublishArticle(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.Empty, error) {
	if err := s.app.PublishArticle(ctx, req.ArticleId, req.UserId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *ContentServer) RecoverArticle(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.Empty, error) {
	if err := s.app.RecoverArticle(ctx, req.ArticleId, req.UserId); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *ContentServer) GetPublishedArticle(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.ArticleDetail, error) {
	detail, err := s.app.GetPublishedArticle(ctx, req.ArticleId, req.UserId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return toArticleDetail(detail), nil
}

func (s *ContentServer) GetArticle(ctx context.Context, req *internalv1.ArticleUserRequest) (*internalv1.ArticleDetail, error) {
	detail, err := s.app.GetArticle(ctx, req.ArticleId, req.UserId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return toArticleDetail(detail), nil
}

func (s *ContentServer) GetPublishedList(ctx context.Context, req *internalv1.ListArticlesRequest) (*internalv1.ArticleList, error) {
	list, err := s.app.GetPublishedList(ctx, req.Page, req.PageSize, req.LastId, req.IsDesc)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return toArticleList(list), nil
}

func (s *ContentServer) GetAdminList(ctx context.Context, req *internalv1.AdminListRequest) (*internalv1.AdminList, error) {
	list, err := s.app.GetAdminList(ctx, req.Page, req.PageSize, req.LastId, req.IsDesc, int8(req.Status))
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return toAdminList(list), nil
}

func (s *ContentServer) GetAvailableList(ctx context.Context, req *internalv1.ListArticlesRequest) (*internalv1.AvailableList, error) {
	list, err := s.app.GetAvailableList(ctx, req.Page, req.PageSize, req.IsDesc)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	items := make([]*internalv1.AvailableItem, 0, len(list.List))
	for _, item := range list.List {
		items = append(items, &internalv1.AvailableItem{
			Id:          item.ID,
			Title:       item.Title,
			Tags:        item.Tags,
			CreatedTime: item.CreatedTime,
			UpdatedTime: item.UpdatedTime,
		})
	}
	return &internalv1.AvailableList{
		Items:    items,
		Total:    list.Total,
		Page:     list.Page,
		PageSize: list.PageSize,
	}, nil
}

func (s *ContentServer) GetImageUploadURL(ctx context.Context, req *internalv1.ImageUploadRequest) (*internalv1.UploadURLResponse, error) {
	uploadURL, url, err := s.app.GetUploadURL(ctx, req.FileExt)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.UploadURLResponse{UploadUrl: uploadURL, Url: url}, nil
}

func (s *ContentServer) GetArticleInfo(ctx context.Context, req *internalv1.ArticleIDRequest) (*internalv1.ArticleInfo, error) {
	article, err := s.app.GetArticleInfo(ctx, req.ArticleId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.ArticleInfo{
		Id:       article.ID,
		AuthorId: article.AuthorID,
		Title:    article.Title,
	}, nil
}

func toArticleDetail(detail *articledto.ArticleDetailResponse) *internalv1.ArticleDetail {
	return &internalv1.ArticleDetail{
		Id:           detail.ID,
		Title:        detail.Title,
		Content:      detail.Content,
		Tags:         detail.Tags,
		Status:       int32(detail.Status),
		AuthorNick:   detail.AuthorNick,
		AuthorAvatar: detail.AuthorAvatar,
		Ip:           detail.IP,
		CreatedTime:  detail.CreatedTime,
		UpdatedTime:  detail.UpdatedTime,
		IsLiked:      detail.IsLiked,
		LikeCount:    detail.LikeCount,
	}
}

func toArticleList(list *articledto.ArticleListResponse) *internalv1.ArticleList {
	items := make([]*internalv1.ArticleListItem, 0, len(list.List))
	for _, item := range list.List {
		items = append(items, &internalv1.ArticleListItem{
			Id:           item.ID,
			Title:        item.Title,
			Summary:      item.Summary,
			AuthorId:     item.AuthorID,
			UpdatedTime:  item.UpdatedTime,
			ViewCount:    item.ViewCount,
			LikeCount:    item.LikeCount,
			CommentCount: item.CommentCount,
		})
	}
	return &internalv1.ArticleList{
		Items:    items,
		LastId:   list.LastID,
		Total:    list.Total,
		Page:     list.Page,
		PageSize: list.PageSize,
	}
}

func toAdminList(list *articledto.AdminListResponse) *internalv1.AdminList {
	items := make([]*internalv1.AdminListItem, 0, len(list.List))
	for _, item := range list.List {
		items = append(items, &internalv1.AdminListItem{
			Id:          item.ID,
			Title:       item.Title,
			Tags:        item.Tags,
			Status:      int32(item.Status),
			CreatedTime: item.CreatedTime,
			UpdatedTime: item.UpdatedTime,
		})
	}
	return &internalv1.AdminList{
		Items:    items,
		LastId:   list.LastID,
		Total:    list.Total,
		Page:     list.Page,
		PageSize: list.PageSize,
	}
}

func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}
	return strings.Split(tags, ",")
}
