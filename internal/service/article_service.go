package service

import (
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
	"time"
)

type ArticleService struct {
	repo *repository.ArticleRepository
}

func NewArticleService(repo *repository.ArticleRepository) *ArticleService {
	return &ArticleService{repo: repo}
}

// 创建文章,创建的文章可能是草稿或者发表的
func (s *ArticleService) CreateArticle(article *model.Article) error {
	now := time.Now()
	article.AddTime = now
	article.UpdateTime = now
	return s.repo.CreateArticle(article)
}

// 更新文章,更新的文章可能是草稿或者发表的
func (s *ArticleService) UpdateArticle(article *model.Article) error {
	OldArticle, err := s.repo.FindArticleByID(article.ID)
	if err != nil {
		return err
	}
	if OldArticle.Status == model.Deleted {
		return common.ErrArticleDeleted
	}
	if OldArticle.AuthorID != article.AuthorID {
		return common.ErrArticlePermissionDenied
	}
	return s.repo.UpdateArticle(article)
}

// 软删除文章
func (s *ArticleService) DeleteArticle(articleId int64, userId int64) error {
	return s.repo.DeleteArticle(articleId, userId)
}

// 硬删除文章
func (s *ArticleService) ClearArticle(articleId int64, userId int64) error {
	return s.repo.ClearArticle(articleId, userId)
}

// 查看文章详情
func (s *ArticleService) GetArticle(articleId int64) (*model.Article, error) {
	article, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, common.ErrArticleNotFound
	}
	if article.Status == model.Deleted {
		return nil, common.ErrArticleDeleted
	}
	return article, nil
}

// 发表文章
func (s *ArticleService) PublishArticle(articleId int64, userId int64) error {
	article, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		return err
	}
	if article.Status == model.Deleted {
		return common.ErrArticleDeleted
	}
	article.AuthorID = userId
	article.Status = model.Published
	article.UpdateTime = time.Now()
	return s.repo.UpdateArticle(article)
}

// 恢复文章
func (s *ArticleService) RecoverArticle(articleId int64, userId int64) error {
	article, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		return err
	}
	article.Status = model.Draft
	article.AuthorID = userId
	article.UpdateTime = time.Now()
	return s.repo.UpdateArticle(article)
}

// 获取已发表文章列表
func (s *ArticleService) GetPublishedList(authorID int64) ([]*model.Article, error) {
	return s.repo.GetListByStatus(authorID, model.Published)
}

// 管理者：获取文章列表
func (s *ArticleService) GetList(authorID int64, status int8) ([]*model.Article, error) {
	return s.repo.GetListByStatus(authorID, status)
}
