package content

import (
	domaincontent "blog/internal/domain/content"
	"blog/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type articleRepository struct {
	db *gorm.DB
}

// NewArticleRepository 返回直接持有 GORM 的 Content 文章 Repository 实现。
func NewArticleRepository(db *gorm.DB) domaincontent.ArticleRepository {
	return &articleRepository{db: db}
}

func (r *articleRepository) Create(ctx context.Context, article *domaincontent.Article) error {
	m := toModelArticle(article)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	article.ID = m.ID
	article.CreatedTime = m.CreatedTime
	article.UpdatedTime = m.UpdatedTime
	return nil
}

func (r *articleRepository) FindByID(ctx context.Context, id uint64) (*domaincontent.Article, error) {
	var m model.Article
	err := r.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time").
		Where("id=?", id).
		Take(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincontent.ErrArticleNotFound
		}
		return nil, err
	}
	return toDomainArticle(&m), nil
}

func (r *articleRepository) FindWithAuthorByID(ctx context.Context, id uint64) (*domaincontent.ArticleWithAuthor, error) {
	article, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domaincontent.ArticleWithAuthor{Article: *article}, nil
}

func (r *articleRepository) Update(ctx context.Context, article *domaincontent.Article) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id=?", article.ID).
		Select("title", "content", "status", "tags").
		Updates(toModelArticle(article)).Error
}

func (r *articleRepository) SoftDelete(ctx context.Context, articleID uint64) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id=?", articleID).
		Updates(map[string]any{"status": model.Deleted}).Error
}

func (r *articleRepository) Clear(ctx context.Context, articleID uint64) error {
	return r.db.WithContext(ctx).Table("articles").
		Where("id=?", articleID).
		Delete(nil).Error
}

func (r *articleRepository) ListWithCursor(ctx context.Context, lastID uint64, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	tx := r.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time")
	tx = applyStatusCondition(tx, status)
	if isDesc {
		tx = tx.Where("id < ?", lastID).Order("id DESC")
	} else {
		tx = tx.Where("id > ?", lastID).Order("id ASC")
	}
	var models []*model.Article
	if err := tx.Limit(pageSize).Find(&models).Error; err != nil {
		return nil, err
	}
	return toDomainArticles(models), nil
}

func (r *articleRepository) ListWithOffset(ctx context.Context, page, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	tx := r.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time")
	tx = applyStatusCondition(tx, status)
	if isDesc {
		tx = tx.Order("id DESC")
	} else {
		tx = tx.Order("id ASC")
	}
	var models []*model.Article
	if err := tx.Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return nil, err
	}
	return toDomainArticles(models), nil
}

func (r *articleRepository) CountByStatus(ctx context.Context, status int8) (int64, error) {
	var count int64
	tx := r.db.WithContext(ctx).Model(&model.Article{})
	tx = applyStatusCondition(tx, status)
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func applyStatusCondition(tx *gorm.DB, status int8) *gorm.DB {
	switch status {
	case domaincontent.StatusAll:
		return tx
	case domaincontent.StatusAllExceptDeleted:
		return tx.Where("status IN ?", []int8{domaincontent.StatusDraft, domaincontent.StatusPublished})
	default:
		return tx.Where("status = ?", status)
	}
}

func toDomainArticle(m *model.Article) *domaincontent.Article {
	return &domaincontent.Article{
		ID:           m.ID,
		AuthorID:     m.AuthorID,
		Title:        m.Title,
		Content:      m.Content,
		Tags:         m.Tags,
		Status:       m.Status,
		ViewCount:    m.ViewCount,
		LikeCount:    m.LikeCount,
		CommentCount: m.CommentCount,
		CreatedTime:  m.CreatedTime,
		UpdatedTime:  m.UpdatedTime,
	}
}

func toModelArticle(a *domaincontent.Article) *model.Article {
	return &model.Article{
		ID:           a.ID,
		AuthorID:     a.AuthorID,
		Title:        a.Title,
		Content:      a.Content,
		Tags:         a.Tags,
		Status:       a.Status,
		ViewCount:    a.ViewCount,
		LikeCount:    a.LikeCount,
		CommentCount: a.CommentCount,
		CreatedTime:  a.CreatedTime,
		UpdatedTime:  a.UpdatedTime,
	}
}

func toDomainArticles(models []*model.Article) []*domaincontent.Article {
	articles := make([]*domaincontent.Article, 0, len(models))
	for _, m := range models {
		articles = append(articles, toDomainArticle(m))
	}
	return articles
}
