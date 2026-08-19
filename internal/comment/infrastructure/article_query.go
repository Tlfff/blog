package infrastructure

import (
	domain "blog/internal/comment/domain"
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

func (q *articleQuery) Exists(ctx context.Context, articleID uint64) (bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := q.client.GetArticleInfo(callCtx, &internalv1.ArticleIDRequest{ArticleId: articleID})
	if err != nil {
		return false, err
	}
	return true, nil
}
