package infrastructure

import (
	domain "blog/internal/notification/domain"
	internalv1 "blog/shared/contracts/gen/internalv1"
	"context"
	"time"
)

type articleQuery struct {
	client internalv1.ContentServiceClient
}

func NewArticleQuery(client internalv1.ContentServiceClient) domain.ArticleQuery {
	return &articleQuery{client: client}
}
func (q *articleQuery) FindByID(ctx context.Context, id uint64) (*domain.ArticleInfo, error) {
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := q.client.GetArticleInfo(callCtx, &internalv1.ArticleIDRequest{ArticleId: id})
	if err != nil {
		return nil, err
	}
	return &domain.ArticleInfo{ID: resp.Id, AuthorID: resp.AuthorId, Title: resp.Title}, nil
}

type userInfoQuery struct {
	client internalv1.IdentityServiceClient
}

func NewUserInfoQuery(client internalv1.IdentityServiceClient) domain.UserInfoQuery {
	return &userInfoQuery{client: client}
}
func (q *userInfoQuery) FindUserByID(ctx context.Context, id uint64) (*domain.UserInfo, error) {
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := q.client.GetUserBasicInfo(callCtx, &internalv1.UserIDRequest{UserId: id})
	if err != nil {
		return nil, err
	}
	return &domain.UserInfo{ID: resp.Id, Nickname: resp.Nickname, Avatar: resp.Avatar}, nil
}
