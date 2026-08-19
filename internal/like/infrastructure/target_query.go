package infrastructure

import (
	commentdomain "blog/internal/comment/domain"
	domain "blog/internal/like/domain"
	internalv1 "blog/shared/contracts/gen/internalv1"
	"context"
	"time"
)

type targetQuery struct {
	articles internalv1.ContentServiceClient
	comments commentdomain.CommentRepository
}

func NewTargetQuery(articles internalv1.ContentServiceClient, comments commentdomain.CommentRepository) domain.TargetQuery {
	return &targetQuery{articles: articles, comments: comments}
}

func (q *targetQuery) ArticleExists(ctx context.Context, articleID uint64) (bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := q.articles.GetArticleInfo(callCtx, &internalv1.ArticleIDRequest{ArticleId: articleID})
	return err == nil, err
}

func (q *targetQuery) CommentExists(ctx context.Context, commentID uint64) (bool, error) {
	comment, err := q.comments.FindByID(ctx, commentID)
	if err != nil {
		return false, err
	}
	return comment.IsNormal(), nil
}
