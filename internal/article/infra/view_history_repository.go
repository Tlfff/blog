package infra

import (
	domaincommunity "blog/internal/article/domain"
	"blog/internal/article/infra/model"
	"context"

	"gorm.io/gorm"
)

// viewHistoryRepository 是浏览历史 Repository 的 GORM 实现。
type viewHistoryRepository struct {
	db *gorm.DB // GORM 数据库连接
}

// NewViewHistoryRepository 返回直接持有 GORM 的浏览历史 Repository 实现。
func NewViewHistoryRepository(db *gorm.DB) domaincommunity.ViewHistoryRepository {
	return &viewHistoryRepository{db: db}
}

// Create 新增一条浏览历史记录。
func (r *viewHistoryRepository) Create(ctx context.Context, history *domaincommunity.ViewHistory) error {
	return r.db.WithContext(ctx).Create(&model.ArticleViewHistory{
		UserID:      history.UserID,
		ArticleID:   history.ArticleID,
		CreatedTime: history.CreatedTime,
		UpdatedTime: history.UpdatedTime,
	}).Error
}

// IncrementViewCount 原子增加文章浏览量。
func (r *viewHistoryRepository) IncrementViewCount(ctx context.Context, articleID uint64) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", articleID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}
