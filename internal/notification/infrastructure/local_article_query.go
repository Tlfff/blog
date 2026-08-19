package infrastructure

import (
	articlemodel "blog/internal/article/infrastructure/model"
	domain "blog/internal/notification/domain"
	"context"

	"gorm.io/gorm"
)

type localArticleQuery struct{ db *gorm.DB }

func NewLocalArticleQuery(db *gorm.DB) domain.ArticleQuery { return &localArticleQuery{db: db} }

func (q *localArticleQuery) FindByID(ctx context.Context, id uint64) (*domain.ArticleInfo, error) {
	var article articlemodel.Article
	if err := q.db.WithContext(ctx).Select("id,author_id,title").Where("id = ?", id).First(&article).Error; err != nil {
		return nil, err
	}
	return &domain.ArticleInfo{ID: article.ID, AuthorID: article.AuthorID, Title: article.Title}, nil
}
