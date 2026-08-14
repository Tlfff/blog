package service

import (
	"blog/internal/common"
	"blog/internal/dto/article"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/oss"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 匹配正文中的图片 URL，排除 Markdown 链接结束符 ) 和引号等
var articleImageURLRe = regexp.MustCompile(`https?://[^\s"'<>()]+`)

type ArticleService struct {
	repo            *repository.ArticleRepository
	userRepo        *repository.UserRepository
	rdb             *redis.Client
	artLikeService  *ArticleLikeService
	oss             *oss.MinioClient
	ossPublicDomain string
}

func NewArticleService(repo *repository.ArticleRepository, artLikeService *ArticleLikeService, rdb *redis.Client) *ArticleService {
	return &ArticleService{
		repo:           repo,
		artLikeService: artLikeService,
		rdb:            rdb,
	}
}

// SetOSS 注入 MinIO 客户端和公开域名（用于文章图片转正）
func (s *ArticleService) SetOSS(ossClient *oss.MinioClient, publicDomain string) {
	s.oss = ossClient
	s.ossPublicDomain = publicDomain
}

// 把正文中 article/temp/ 目录下的图片转正到 article/{article_id}/ 目录，
// 并同步替换正文中的 URL。Copy 成功前不删除源，失败可安全重试。
func (s *ArticleService) PromoteImages(ctx context.Context, articleID uint64, content string) (string, error) {
	// 1. 检查是否配置了 OSS 客户端
	if s.oss == nil || content == "" {
		return content, nil
	}

	seen := make(map[string]bool)
	var promoteErr error
	// 2. 遍历正文中的所有图片 URL
	for _, rawURL := range articleImageURLRe.FindAllString(content, -1) {
		// 2.1 只处理本站公开域名下的图片
		if !strings.HasPrefix(rawURL, s.ossPublicDomain) {
			continue
		}
		// 2.2 去掉 query 参数
		u := strings.SplitN(rawURL, "?", 2)[0]
		srcKey := strings.TrimPrefix(u, s.ossPublicDomain+"/")
		// 2.3 只处理 temp 目录下的图片
		if !strings.HasPrefix(srcKey, "article/temp/") {
			continue
		}
		// 2.4 避免重复处理
		if seen[srcKey] {
			continue
		}
		seen[srcKey] = true

		// 2.5 转正：article/temp/xxx.ext -> article/{article_id}/xxx.ext（按文章隔离）
		dstKey := "article/" + fmt.Sprint(articleID) + "/" + strings.TrimPrefix(srcKey, "article/temp/")
		if err := s.oss.MoveObject(ctx, srcKey, dstKey); err != nil {
			if promoteErr == nil {
				promoteErr = err
			}
			continue
		}
		// 同步替换正文中的 URL
		srcURL := s.ossPublicDomain + "/" + srcKey
		dstURL := s.ossPublicDomain + "/" + dstKey
		content = strings.ReplaceAll(content, srcURL, dstURL)
	}

	return content, promoteErr
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

	// 先落库拿到自增文章ID（图片转正按文章隔离，需要文章ID）
	if err := s.repo.CreateArticle(ctx, art); err != nil {
		return err
	}

	// 保存（草稿或发表）时把正文中 temp 目录的图片转正
	newContent, err := s.PromoteImages(ctx, art.ID, content)
	if err != nil {
		// 转正失败则删除刚创建的文章，避免留下引用 temp 图片的脏数据
		_ = s.repo.ClearArticle(ctx, art.ID, authorID)
		return err
	}
	if newContent != content {
		art.Content = newContent
		return s.repo.UpdateArticle(ctx, art)
	}
	return nil
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

	// 保存（草稿或发表）时把正文中 temp 目录的图片转正
	content, err = s.PromoteImages(ctx, articleId, content)
	if err != nil {
		return err
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
func (s *ArticleService) GetPublishedArticle(ctx context.Context, articleId uint64, userId uint64) (*article.ArticleDetailResponse, error) {

	// 1 查出文章
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

	// 2. 判断当前登录用户是否点赞过该文章
	var isLiked bool
	if userId > 0 {
		isLiked, _ = s.artLikeService.IsUserLikedArticle(ctx, userId, articleId)
	}

	// 3. 构造详情响应
	resp := article.NewArticleDetailResponse(&detail.Article, detail.Nickname, detail.Avatar, detail.LastLoginIp, isLiked)

	return resp, nil
}

// 管理员：查看文章详情
func (s *ArticleService) GetArticle(ctx context.Context, articleId, userId uint64) (*article.ArticleDetailResponse, error) {
	// 1.获取文章详情
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
	// 2. 判断当前登录用户是否点赞过该文章
	var isLiked bool
	if userId > 0 {
		isLiked, _ = s.artLikeService.IsUserLikedArticle(ctx, userId, articleId)
	}

	// 3. 构造详情响应
	resp := article.NewArticleDetailResponse(&detail.Article, detail.Nickname, detail.Avatar, detail.LastLoginIp, isLiked)

	return resp, nil
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

	// 发布只改状态，不修改内容；正文中的 temp 图片在保存草稿/更新文章时已转正
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
	return article.NewArticleListResponse(list, uint64(total), nextLastID, page, pageSize), nil
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

	return article.NewAdminListResponse(list, uint64(total), nextLastID, page, pageSize), nil
}

// ------------------------------------ 二方 ------------------------------------

// 获取可用文章列表
func (s *ArticleService) GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*article.ExternalListResponse, error) {
	var list []*model.Article
	var err error

	// 1. 传统分页
	list, err = s.repo.GetListWithOffset(ctx, int(page), int(pageSize), isDesc, model.Published)
	if err != nil {
		return nil, err
	}
	// 2. 计算发表的总文章数
	total, err := s.repo.GetArticleCountByStatus(ctx, model.Published)
	if err != nil {
		return nil, err
	}
	return article.NewExternalListResponse(list, uint64(total), page, pageSize), nil
}
