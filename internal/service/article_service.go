package service

import (
	"blog/internal/common"
	"blog/internal/dto/article"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
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
func (s *ArticleService) CreateArticle(ctx context.Context, authorID uint64, title, content string, tags []string, status int8) error {

	// 手动拼接标签 ["Go", "Gin"] -> "Go,Gin"
	tagsStr := strings.Join(tags, ",")

	art := &model.Article{
		AuthorID: authorID,
		Title:    title,
		Content:  content,
		Tags:     tagsStr,
		Status:   status,
	}

	return s.repo.CreateArticle(ctx, art)
}

// 更新文章,更新的文章可能是草稿或者发表的
func (s *ArticleService) UpdateArticle(ctx context.Context, articleId uint64, authorID uint64, title, content string, tags []string, status int8) error {
	// 鉴权：先查出老文章
	oldArticle, err := s.repo.FindArticleByID(ctx, articleId)
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

	return s.repo.UpdateArticle(ctx, art)
}

// 软删除文章
func (s *ArticleService) DeleteArticle(ctx context.Context, articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(ctx, articleId)
	if err != nil {
		return err
	}
	if oldArticle.AuthorID != userId {
		return common.ErrArticlePermissionDenied
	}
	return s.repo.DeleteArticle(ctx, articleId)
}

// 硬删除文章
func (s *ArticleService) ClearArticle(ctx context.Context, articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(ctx, articleId)
	if err != nil {
		return err
	}
	if oldArticle.AuthorID != userId {
		return common.ErrArticlePermissionDenied
	}
	return s.repo.ClearArticle(ctx, articleId, userId)
}

// 公开：查看文章详情
func (s *ArticleService) GetPublishedArticle(ctx context.Context, articleId uint64, userId uint64, ip string) (*article.ArticleDetailResponse, error) {
	// 1.查出文章
	detail, err := s.repo.FindArticleAndUserInfoByID(ctx, articleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrArticleNotFound
		}
		return nil, err
	}
	if detail.Status == model.Deleted {
		return nil, common.ErrArticleDeleted
	}
	if detail.Status != model.Published {
		return nil, common.ErrArticlePermissionDenied
	}

	// 2.记录浏览历史
	s.historyView.RecordView(ctx, userId, articleId, ip)

	return article.NewArticleDetailResponse(&detail.Article, detail.Nickname, detail.Avatar, detail.LastLoginIp), nil
}

// 管理员：查看文章详情
func (s *ArticleService) GetArticle(ctx context.Context, articleId uint64) (*article.ArticleDetailResponse, error) {
	detail, err := s.repo.FindArticleAndUserInfoByID(ctx, articleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrArticleNotFound
		}
		return nil, err
	}
	if detail.Status == model.Deleted {
		return nil, common.ErrArticleDeleted
	}

	return article.NewArticleDetailResponse(&detail.Article, detail.Nickname, detail.Avatar, detail.LastLoginIp), nil
}

// 发表文章
func (s *ArticleService) PublishArticle(ctx context.Context, articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(ctx, articleId)
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
	return s.repo.UpdateArticle(ctx, oldArticle)
}

// 恢复文章
func (s *ArticleService) RecoverArticle(ctx context.Context, articleId uint64, userId uint64) error {
	oldArticle, err := s.repo.FindArticleByID(ctx, articleId)
	if err != nil {
		return err
	}
	oldArticle.Status = model.Draft
	oldArticle.AuthorID = userId
	return s.repo.UpdateArticle(ctx, oldArticle)
}

// 获取已发表文章列表
func (s *ArticleService) GetPublishedList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool) (*article.ArticleListResponse, error) {
	var list []*model.Article
	var err error
	if lastID > 0 {
		// 1. 如果有lastID，则用游标分页方式获取
		list, err = s.repo.GetListWithCursor(ctx, lastID, int(pageSize), isDesc, model.Published)

	} else {
		// 2. 否则用传统分页
		list, err = s.repo.GetListWithOffset(ctx, int(page), int(pageSize), isDesc, model.Published)
	}
	// 3. 计算发表的总文章数
	total, err := s.repo.GetArticleCountByStatus(ctx, model.Published)
	if err != nil {
		return nil, err
	}
	// 4.获取当页的最后一个id
	nextLastID := uint64(0)
	if len(list) > 0 {
		nextLastID = list[len(list)-1].ID
	}
	return article.NewArticleListResponse(list, uint64(total), nextLastID), nil
}

// 管理者：获取文章列表
func (s *ArticleService) GetAdminList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool, status int8) (*article.AdminListResponse, error) {
	var list []*model.Article
	var err error
	if lastID > 0 {
		// 1. 如果有lastID，则用游标分页方式获取
		list, err = s.repo.GetListWithCursor(ctx, lastID, int(pageSize), isDesc, status)

	} else {
		// 2. 否则用传统分页
		list, err = s.repo.GetListWithOffset(ctx, int(page), int(pageSize), isDesc, status)
	}
	// 3. 计算发表的总文章数
	total, err := s.repo.GetArticleCountByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	// 4.获取当页的最后一个id
	nextLastID := uint64(0)
	if len(list) > 0 {
		nextLastID = list[len(list)-1].ID
	}

	return article.NewAdminListResponse(list, uint64(total), nextLastID), nil
}
