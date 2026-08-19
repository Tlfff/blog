package client

import (
	articledto "blog/internal/article/application/dto"
	"blog/internal/shared/common"
	internalv1 "blog/shared/contracts/gen/internalv1"
	platformerrors "blog/shared/platform/errors"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ContentClient 实现统一入口需要的 Content 用例接口。
type ContentClient struct {
	client internalv1.ContentServiceClient
}

func NewContentClient(client internalv1.ContentServiceClient) *ContentClient {
	return &ContentClient{client: client}
}

func (c *ContentClient) CreateArticle(ctx context.Context, authorID uint64, title, content string, tags []string, status int8) error {
	_, err := c.client.CreateArticle(ctx, &internalv1.CreateArticleRequest{
		AuthorId: authorID,
		Title:    title,
		Content:  content,
		Tags:     tags,
		Status:   int32(status),
	})
	return toContentError(err)
}

func (c *ContentClient) UpdateArticle(ctx context.Context, articleID, authorID uint64, title, content string, tags []string, status int8) error {
	_, err := c.client.UpdateArticle(ctx, &internalv1.UpdateArticleRequest{
		ArticleId: articleID,
		AuthorId:  authorID,
		Title:     title,
		Content:   content,
		Tags:      tags,
		Status:    int32(status),
	})
	return toContentError(err)
}

func (c *ContentClient) DeleteArticle(ctx context.Context, articleID, userID uint64) error {
	_, err := c.client.DeleteArticle(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	return toContentError(err)
}

func (c *ContentClient) ClearArticle(ctx context.Context, articleID, userID uint64) error {
	_, err := c.client.ClearArticle(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	return toContentError(err)
}

func (c *ContentClient) PublishArticle(ctx context.Context, articleID, userID uint64) error {
	_, err := c.client.PublishArticle(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	return toContentError(err)
}

func (c *ContentClient) RecoverArticle(ctx context.Context, articleID, userID uint64) error {
	_, err := c.client.RecoverArticle(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	return toContentError(err)
}

func (c *ContentClient) GetPublishedArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	resp, err := c.client.GetPublishedArticle(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	if err != nil {
		return nil, toContentError(err)
	}
	return toArticleDetailDTO(resp), nil
}

func (c *ContentClient) GetArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	resp, err := c.client.GetArticle(ctx, &internalv1.ArticleUserRequest{ArticleId: articleID, UserId: userID})
	if err != nil {
		return nil, toContentError(err)
	}
	return toArticleDetailDTO(resp), nil
}

func (c *ContentClient) GetPublishedList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool) (*articledto.ArticleListResponse, error) {
	resp, err := c.client.GetPublishedList(ctx, &internalv1.ListArticlesRequest{
		Page:     page,
		PageSize: pageSize,
		LastId:   lastID,
		IsDesc:   isDesc,
	})
	if err != nil {
		return nil, toContentError(err)
	}
	list := &articledto.ArticleListResponse{
		List:     make([]*articledto.ArticleListItem, 0, len(resp.Items)),
		LastID:   resp.LastId,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}
	for _, item := range resp.Items {
		list.List = append(list.List, &articledto.ArticleListItem{
			ID:           item.Id,
			Title:        item.Title,
			Summary:      item.Summary,
			AuthorID:     item.AuthorId,
			UpdatedTime:  item.UpdatedTime,
			ViewCount:    item.ViewCount,
			LikeCount:    item.LikeCount,
			CommentCount: item.CommentCount,
		})
	}
	return list, nil
}

func (c *ContentClient) GetAdminList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool, status int8) (*articledto.AdminListResponse, error) {
	resp, err := c.client.GetAdminList(ctx, &internalv1.AdminListRequest{
		Page:     page,
		PageSize: pageSize,
		LastId:   lastID,
		IsDesc:   isDesc,
		Status:   int32(status),
	})
	if err != nil {
		return nil, toContentError(err)
	}
	list := &articledto.AdminListResponse{
		List:     make([]*articledto.AdminListItem, 0, len(resp.Items)),
		LastID:   resp.LastId,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}
	for _, item := range resp.Items {
		list.List = append(list.List, &articledto.AdminListItem{
			ID:          item.Id,
			Title:       item.Title,
			Tags:        item.Tags,
			Status:      int8(item.Status),
			CreatedTime: item.CreatedTime,
			UpdatedTime: item.UpdatedTime,
		})
	}
	return list, nil
}

func (c *ContentClient) GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*articledto.ExternalListResponse, error) {
	resp, err := c.client.GetAvailableList(ctx, &internalv1.ListArticlesRequest{
		Page:     page,
		PageSize: pageSize,
		IsDesc:   isDesc,
	})
	if err != nil {
		return nil, toContentError(err)
	}
	list := &articledto.ExternalListResponse{
		List:     make([]*articledto.ExternalListItem, 0, len(resp.Items)),
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}
	for _, item := range resp.Items {
		list.List = append(list.List, &articledto.ExternalListItem{
			ID:          item.Id,
			Title:       item.Title,
			Tags:        item.Tags,
			CreatedTime: item.CreatedTime,
			UpdatedTime: item.UpdatedTime,
		})
	}
	return list, nil
}

func (c *ContentClient) GetUploadURL(ctx context.Context, fileExt string) (uploadURL, url string, err error) {
	resp, err := c.client.GetImageUploadURL(ctx, &internalv1.ImageUploadRequest{FileExt: fileExt})
	if err != nil {
		return "", "", toContentError(err)
	}
	return resp.UploadUrl, resp.Url, nil
}

func toArticleDetailDTO(resp *internalv1.ArticleDetail) *articledto.ArticleDetailResponse {
	return &articledto.ArticleDetailResponse{
		ID:           resp.Id,
		Title:        resp.Title,
		Content:      resp.Content,
		Tags:         resp.Tags,
		Status:       int8(resp.Status),
		AuthorNick:   resp.AuthorNick,
		AuthorAvatar: resp.AuthorAvatar,
		IP:           resp.Ip,
		CreatedTime:  resp.CreatedTime,
		UpdatedTime:  resp.UpdatedTime,
		IsLiked:      resp.IsLiked,
		LikeCount:    resp.LikeCount,
	}
}

func toContentError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		return common.ErrArticleNotFound
	}
	switch platformerrors.FromGRPC(err) {
	case common.CodeForbidden:
		return common.ErrForbidden
	case common.CodeUnauthorized:
		return common.ErrTokenInvalid
	case common.CodeInvalidParameter:
		return common.ErrParameter
	default:
		return common.ErrSystem
	}
}
