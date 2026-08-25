package infra

import (
	articledomain "blog/internal/article/domain"
	"context"
)

// LikeFacade 是 Article 查询当前用户点赞状态所需的 Like Application Facade。
type LikeFacade interface {
	// IsUserLikedArticle 查询用户是否点赞文章。
	IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error)
}

type interactionQuery struct {
	facade LikeFacade // Like Application Facade
}

// NewInteractionQuery 创建 Article 到 Like 的本地查询适配器。
func NewInteractionQuery(facade LikeFacade) articledomain.ArticleInteractionQuery {
	return &interactionQuery{facade: facade}
}

// IsUserLikedArticle 查询用户文章点赞状态。
func (q *interactionQuery) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	return q.facade.IsUserLikedArticle(ctx, userID, articleID)
}
