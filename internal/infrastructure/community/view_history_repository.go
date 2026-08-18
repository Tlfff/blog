package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
)

type viewHistoryRepository struct {
	db *gorm.DB
}

// NewViewHistoryRepository 返回直接持有 GORM 的浏览历史 Repository 实现。
func NewViewHistoryRepository(db *gorm.DB) domaincommunity.ViewHistoryRepository {
	return &viewHistoryRepository{db: db}
}

func (r *viewHistoryRepository) Create(ctx context.Context, history *domaincommunity.ViewHistory) error {
	return r.db.WithContext(ctx).Create(&model.ArticleViewHistory{
		UserID:      history.UserID,
		ArticleID:   history.ArticleID,
		CreatedTime: history.CreatedTime,
		UpdatedTime: history.UpdatedTime,
	}).Error
}

func (r *viewHistoryRepository) IncrementViewCount(ctx context.Context, articleID uint64) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", articleID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}
