package service

import (
	"blog/internal/common"
	"blog/internal/dto/article"
	"blog/internal/model"
	"blog/internal/repository"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type ArticleService struct {
	repo        *repository.ArticleRepository
	userRepo    *repository.UserRepository
	historyView *ArticleViewHistoryService
}

func NewArticleService(repo *repository.ArticleRepository, historyView *ArticleViewHistoryService) *ArticleService {
	return &ArticleService{
		repo:        repo,
		historyView: historyView,
	}
}

// 创建文章,创建的文章可能是草稿或者发表的
func (s *ArticleService) CreateArticle(authorID uint64, title, content string, tags []string, status int8) error {

	// 手动拼接标签 ["Go", "Gin"] -> "Go,Gin"
	tagsStr := strings.Join(tags, ",")

	art := &model.Article{
		AuthorID: authorID,
		Title:    title,
		Content:  content,
		Tags:     tagsStr,
		Status:   status,
	}

	return s.repo.CreateArticle(art)
}

// 更新文章,更新的文章可能是草稿或者发表的
func (s *ArticleService) UpdateArticle(articleId uint64, authorID uint64, title, content string, tags []string, status int8) error {
	// 鉴权：先查出老文章
	oldArticle, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrArticleNotFound
		}
		return err
	}
	if oldArticle.Status == model.Deleted {
		return common.ErrArticleDeleted
	}
	if oldArticle.AuthorID != authorID {
		return common.ErrArticlePermissionDenied
	}

	tagsStr := strings.Join(tags, ",")

	art := &model.Article{
		ID:      articleId,
		Title:   title,
		Content: content,
		Tags:    tagsStr,
		Status:  status,
	}

	return s.repo.UpdateArticle(art)
}

// 软删除文章
func (s *ArticleService) DeleteArticle(articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		return err
	}
	if oldArticle.AuthorID != userId {
		return common.ErrArticlePermissionDenied
	}
	return s.repo.DeleteArticle(articleId)
}

// 硬删除文章
func (s *ArticleService) ClearArticle(articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		return err
	}
	if oldArticle.AuthorID != userId {
		return common.ErrArticlePermissionDenied
	}
	return s.repo.ClearArticle(articleId, userId)
}

// 公开：查看文章详情
func (s *ArticleService) GetPublishedArticle(articleId uint64, userId uint64, ip string) (*article.ArticleDetailResponse, error) {
	// 1.查出文章
	oldArticle, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrArticleNotFound
		}
		return nil, err
	}
	if oldArticle.Status == model.Deleted {
		return nil, common.ErrArticleDeleted
	}
	if oldArticle.Status != model.Published {
		return nil, common.ErrArticlePermissionDenied
	}

	// 2.记录浏览历史
	s.historyView.RecordView(userId, articleId, ip)
	// 3. todo：假定一个作者昵称，如果未来连了用户表，可以用 artModel.AuthorID 去查出来
	authorNick := "博主"

	return article.NewArticleDetailResponse(oldArticle, authorNick), nil
}

// 管理员：查看文章详情
func (s *ArticleService) GetArticle(articleId uint64) (*article.ArticleDetailResponse, error) {
	oldArticle, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrArticleNotFound
		}
		return nil, err
	}
	if oldArticle.Status == model.Deleted {
		return nil, common.ErrArticleDeleted
	}

	// todo：假定一个作者昵称，如果未来连了用户表，可以用 artModel.AuthorID 去查出来
	authorNick := "博主"

	return article.NewArticleDetailResponse(oldArticle, authorNick), nil
}

// 发表文章
func (s *ArticleService) PublishArticle(articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		return err
	}
	if oldArticle.AuthorID != userId {
		return common.ErrArticlePermissionDenied
	}
	if oldArticle.Status == model.Deleted {
		return common.ErrArticleDeleted
	}

	oldArticle.Status = model.Published
	return s.repo.UpdateArticle(oldArticle)
}

// 恢复文章
func (s *ArticleService) RecoverArticle(articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(articleId)
	if err != nil {
		return err
	}
	oldArticle.Status = model.Draft
	oldArticle.AuthorID = userId
	return s.repo.UpdateArticle(oldArticle)
}

// 获取已发表文章列表
func (s *ArticleService) GetPublishedList(authorID uint64) (*article.ArticleListResponse, error) {
	models, err := s.repo.GetListByStatus(authorID, model.Published)
	if err != nil {
		return nil, err
	}
	return article.NewArticleListResponse(models), nil
}

// 管理者：获取文章列表
func (s *ArticleService) GetAdminList(authorID uint64, status int8) (*article.AdminListResponse, error) {
	models, err := s.repo.GetListByStatus(authorID, status)
	if err != nil {
		return nil, err
	}
	return article.NewAdminListResponse(models), nil
}
