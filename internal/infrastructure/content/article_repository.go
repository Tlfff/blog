package content

import (
	domaincontent "blog/internal/domain/content"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"errors"

	"gorm.io/gorm"
)

type articleRepositoryAdapter struct {
	repo *repository.ArticleRepository
}

// NewArticleRepository 将现有 GORM ArticleRepository 适配为 Content 领域 Port。
func NewArticleRepository(repo *repository.ArticleRepository) domaincontent.ArticleRepository {
	return &articleRepositoryAdapter{repo: repo}
}

func (a *articleRepositoryAdapter) Create(ctx context.Context, article *domaincontent.Article) error {
	m := toModelArticle(article)
	if err := a.repo.CreateArticle(ctx, m); err != nil {
		return err
	}
	article.ID = m.ID
	article.CreatedTime = m.CreatedTime
	article.UpdatedTime = m.UpdatedTime
	return nil
}

func (a *articleRepositoryAdapter) FindByID(ctx context.Context, id uint64) (*domaincontent.Article, error) {
	m, err := a.repo.FindArticleByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincontent.ErrArticleNotFound
		}
		return nil, err
	}
	return toDomainArticle(m), nil
}

func (a *articleRepositoryAdapter) FindWithAuthorByID(ctx context.Context, id uint64) (*domaincontent.ArticleWithAuthor, error) {
	m, err := a.repo.FindArticleAndUserInfoByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincontent.ErrArticleNotFound
		}
		return nil, err
	}
	return &domaincontent.ArticleWithAuthor{
		Article:     *toDomainArticle(&m.Article),
		Nickname:    m.Nickname,
		Avatar:      m.Avatar,
		LastLoginIP: m.LastLoginIp,
	}, nil
}

func (a *articleRepositoryAdapter) Update(ctx context.Context, article *domaincontent.Article) error {
	return a.repo.UpdateArticle(ctx, toModelArticle(article))
}

func (a *articleRepositoryAdapter) SoftDelete(ctx context.Context, articleID uint64) error {
	return a.repo.DeleteArticle(ctx, articleID)
}

func (a *articleRepositoryAdapter) Clear(ctx context.Context, articleID uint64) error {
	return a.repo.ClearArticle(ctx, articleID, 0)
}

func (a *articleRepositoryAdapter) ListWithCursor(ctx context.Context, lastID uint64, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	models, err := a.repo.GetListWithCursor(ctx, lastID, pageSize, isDesc, status)
	if err != nil {
		return nil, err
	}
	return toDomainArticles(models), nil
}

func (a *articleRepositoryAdapter) ListWithOffset(ctx context.Context, page, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	models, err := a.repo.GetListWithOffset(ctx, page, pageSize, isDesc, status)
	if err != nil {
		return nil, err
	}
	return toDomainArticles(models), nil
}

func (a *articleRepositoryAdapter) CountByStatus(ctx context.Context, status int8) (int64, error) {
	return a.repo.GetArticleCountByStatus(ctx, status)
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
