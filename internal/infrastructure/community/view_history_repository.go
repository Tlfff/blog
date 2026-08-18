package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
)

type viewHistoryRepositoryAdapter struct {
	repo *repository.ArticleViewHistoryRepository
}

// NewViewHistoryRepository 将 GORM 浏览历史 Repository 适配为 Community 领域 Port。
func NewViewHistoryRepository(repo *repository.ArticleViewHistoryRepository) domaincommunity.ViewHistoryRepository {
	return &viewHistoryRepositoryAdapter{repo: repo}
}

func (a *viewHistoryRepositoryAdapter) Create(ctx context.Context, history *domaincommunity.ViewHistory) error {
	m := &model.ArticleViewHistory{
		UserID:      history.UserID,
		ArticleID:   history.ArticleID,
		CreatedTime: history.CreatedTime,
		UpdatedTime: history.UpdatedTime,
	}
	return a.repo.CreateViewHistory(ctx, m)
}

func (a *viewHistoryRepositoryAdapter) IncrementViewCount(ctx context.Context, articleID uint64) error {
	return a.repo.IncrementViewCount(ctx, articleID)
}
