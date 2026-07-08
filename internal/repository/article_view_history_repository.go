package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
)

type ArticleViewHistoryRepository struct {
	db *gorm.DB
}

func NewArticleViewHistoryRepository(db *gorm.DB) *ArticleViewHistoryRepository {
	return &ArticleViewHistoryRepository{
		db: db,
	}
}

//	插入一条浏览历史流水
//
// insert into article_view_histories
// (user_id, article_id)
// values (?, ?)
func (a *ArticleViewHistoryRepository) CreateViewHistory(ctx context.Context, history *model.ArticleViewHistory) error {
	return a.db.WithContext(ctx).Create(history).Error
}

// 文章浏览量自增 1
// update article set view_count=view_count + 1 where id =?
func (a *ArticleViewHistoryRepository) IncrementViewCount(ctx context.Context, articleID uint64) error {
	return a.db.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", articleID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error //防止了sql注入，？能否防止丢失
}
