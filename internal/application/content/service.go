// Package content 提供 Content 领域的应用用例。
package content

import (
	"blog/internal/common"
	domaincontent "blog/internal/domain/content"
	articledto "blog/internal/dto/article"
	iputil "blog/pkg/util/ip"
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var articleImageURLRe = regexp.MustCompile(`https?://[^\s"'<>()]+`)

type Service struct {
	repo          domaincontent.ArticleRepository
	users         domaincontent.UserQuery
	images        domaincontent.ArticleImageStorage
	interactions  domaincontent.ArticleInteractionQuery
	publicDomain  string
	allowedExts   map[string]bool
}

// NewService 组装 Content Application 用例依赖。
func NewService(
	repo domaincontent.ArticleRepository,
	users domaincontent.UserQuery,
	images domaincontent.ArticleImageStorage,
	interactions domaincontent.ArticleInteractionQuery,
	publicDomain string,
	allowedExts []string,
) *Service {
	extMap := make(map[string]bool, len(allowedExts))
	for _, ext := range allowedExts {
		extMap[strings.ToLower(ext)] = true
	}
	return &Service{
		repo:         repo,
		users:        users,
		images:       images,
		interactions: interactions,
		publicDomain: publicDomain,
		allowedExts:  extMap,
	}
}

func (s *Service) CreateArticle(ctx context.Context, authorID uint64, title, content string, tags []string, status int8) error {
	art := &domaincontent.Article{
		AuthorID: authorID,
		Title:    title,
		Content:  content,
		Tags:     strings.Join(tags, ","),
		Status:   status,
	}
	if err := s.repo.Create(ctx, art); err != nil {
		return err
	}

	newContent, err := s.PromoteImages(ctx, art.ID, content)
	if err != nil {
		_ = s.repo.Clear(ctx, art.ID)
		return err
	}
	if newContent != content {
		art.Content = newContent
		return s.repo.Update(ctx, art)
	}
	return nil
}

func (s *Service) UpdateArticle(ctx context.Context, articleID, authorID uint64, title, content string, tags []string, status int8) error {
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if oldArticle.IsDeleted() {
		return common.ErrArticleDeleted
	}
	if !oldArticle.CanEdit(authorID) {
		return common.ErrArticlePermissionDenied
	}

	content, err = s.PromoteImages(ctx, articleID, content)
	if err != nil {
		return err
	}

	art := &domaincontent.Article{
		ID:      articleID,
		Title:   title,
		Content: content,
		Tags:    strings.Join(tags, ","),
		Status:  status,
	}
	return s.repo.Update(ctx, art)
}

func (s *Service) DeleteArticle(ctx context.Context, articleID, userID uint64) error {
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if !oldArticle.CanDelete(userID) {
		return common.ErrArticlePermissionDenied
	}
	return s.repo.SoftDelete(ctx, articleID)
}

func (s *Service) ClearArticle(ctx context.Context, articleID, userID uint64) error {
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if !oldArticle.CanDelete(userID) {
		return common.ErrArticlePermissionDenied
	}
	return s.repo.Clear(ctx, articleID)
}

func (s *Service) PublishArticle(ctx context.Context, articleID, userID uint64) error {
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if !oldArticle.CanPublish(userID) {
		return common.ErrArticlePermissionDenied
	}
	if oldArticle.IsDeleted() {
		return common.ErrArticleDeleted
	}
	oldArticle.Publish()
	return s.repo.Update(ctx, oldArticle)
}

func (s *Service) RecoverArticle(ctx context.Context, articleID, userID uint64) error {
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	oldArticle.Recover()
	oldArticle.AuthorID = userID
	return s.repo.Update(ctx, oldArticle)
}

func (s *Service) GetPublishedArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	detail, err := s.findDetail(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if detail.IsDeleted() {
		return nil, common.ErrArticleDeleted
	}
	if !detail.IsPubliclyVisible() {
		return nil, common.ErrArticlePermissionDenied
	}
	return s.buildDetail(ctx, detail, userID)
}

func (s *Service) GetArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	detail, err := s.findDetail(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if detail.IsDeleted() {
		return nil, common.ErrArticleDeleted
	}
	return s.buildDetail(ctx, detail, userID)
}

func (s *Service) GetPublishedList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool) (*articledto.ArticleListResponse, error) {
	var list []*domaincontent.Article
	var err error
	if lastID > 0 {
		list, err = s.repo.ListWithCursor(ctx, lastID, int(pageSize), isDesc, domaincontent.StatusPublished)
	} else {
		list, err = s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, domaincontent.StatusPublished)
	}
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountByStatus(ctx, domaincontent.StatusPublished)
	if err != nil {
		return nil, err
	}
	nextLastID := uint64(0)
	if len(list) > 0 {
		nextLastID = list[len(list)-1].ID
	}
	return newArticleListResponse(list, uint64(total), nextLastID, page, pageSize), nil
}

func (s *Service) GetAdminList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool, status int8) (*articledto.AdminListResponse, error) {
	var list []*domaincontent.Article
	var err error
	if lastID > 0 {
		list, err = s.repo.ListWithCursor(ctx, lastID, int(pageSize), isDesc, status)
	} else {
		list, err = s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, status)
	}
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	nextLastID := uint64(0)
	if len(list) > 0 {
		nextLastID = list[len(list)-1].ID
	}
	return newAdminListResponse(list, uint64(total), nextLastID, page, pageSize), nil
}

func (s *Service) GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*articledto.ExternalListResponse, error) {
	list, err := s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, domaincontent.StatusPublished)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountByStatus(ctx, domaincontent.StatusPublished)
	if err != nil {
		return nil, err
	}
	return newExternalListResponse(list, uint64(total), page, pageSize), nil
}

// GetArticleInfo 返回文章供服务间只读查询使用的基本信息。
func (s *Service) GetArticleInfo(ctx context.Context, articleID uint64) (*domaincontent.Article, error) {
	return s.findArticle(ctx, articleID)
}

func (s *Service) GetUploadURL(ctx context.Context, fileExt string) (uploadURL, url string, err error) {
	if s.images == nil {
		return "", "", common.ErrSystem
	}
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	if !s.allowedExts[ext] {
		return "", "", common.ErrInvalidRequestBody
	}
	objectKey := path.Join("article", "temp", uuid.NewString()+"."+ext)
	uploadURL, err = s.images.PresignedPutURL(ctx, objectKey, 10*time.Minute)
	if err != nil {
		return "", "", err
	}
	return uploadURL, s.images.GetObjectURL(s.publicDomain, objectKey), nil
}

// PromoteImages 把正文中的临时图片转正到文章正式目录。
func (s *Service) PromoteImages(ctx context.Context, articleID uint64, content string) (string, error) {
	if s.images == nil || content == "" {
		return content, nil
	}
	seen := make(map[string]bool)
	var promoteErr error
	for _, rawURL := range articleImageURLRe.FindAllString(content, -1) {
		if !strings.HasPrefix(rawURL, s.publicDomain) {
			continue
		}
		u := strings.SplitN(rawURL, "?", 2)[0]
		srcKey := strings.TrimPrefix(u, s.publicDomain+"/")
		if !strings.HasPrefix(srcKey, "article/temp/") {
			continue
		}
		if seen[srcKey] {
			continue
		}
		seen[srcKey] = true

		dstKey := "article/" + fmt.Sprint(articleID) + "/" + strings.TrimPrefix(srcKey, "article/temp/")
		if err := s.images.MoveObject(ctx, srcKey, dstKey); err != nil {
			if promoteErr == nil {
				promoteErr = err
			}
			continue
		}
		srcURL := s.publicDomain + "/" + srcKey
		dstURL := s.publicDomain + "/" + dstKey
		content = strings.ReplaceAll(content, srcURL, dstURL)
	}
	return content, promoteErr
}

func (s *Service) buildDetail(ctx context.Context, detail *domaincontent.ArticleWithAuthor, userID uint64) (*articledto.ArticleDetailResponse, error) {
	if s.users != nil {
		if user, err := s.users.FindUserByID(ctx, detail.AuthorID); err == nil {
			detail.Nickname = user.Nickname
			detail.Avatar = user.Avatar
			detail.LastLoginIP = user.LastLoginIP
		}
	}
	isLiked := false
	if userID > 0 && s.interactions != nil {
		isLiked, _ = s.interactions.IsUserLikedArticle(ctx, userID, detail.ID)
	}
	return newArticleDetailResponse(detail, isLiked), nil
}

func (s *Service) findArticle(ctx context.Context, articleID uint64) (*domaincontent.Article, error) {
	article, err := s.repo.FindByID(ctx, articleID)
	if errors.Is(err, domaincontent.ErrArticleNotFound) {
		return nil, common.ErrArticleNotFound
	}
	return article, err
}

func (s *Service) findDetail(ctx context.Context, articleID uint64) (*domaincontent.ArticleWithAuthor, error) {
	detail, err := s.repo.FindWithAuthorByID(ctx, articleID)
	if errors.Is(err, domaincontent.ErrArticleNotFound) {
		return nil, common.ErrArticleNotFound
	}
	return detail, err
}

func newArticleDetailResponse(d *domaincontent.ArticleWithAuthor, isLiked bool) *articledto.ArticleDetailResponse {
	tags := splitTags(d.Tags)
	return &articledto.ArticleDetailResponse{
		ID:           d.ID,
		Title:        d.Title,
		Content:      d.Content,
		Tags:         tags,
		Status:       d.Status,
		AuthorNick:   d.Nickname,
		AuthorAvatar: d.Avatar,
		IP:           iputil.ConvertIPToRegion(d.LastLoginIP),
		CreatedTime:  d.CreatedTime.Unix(),
		UpdatedTime:  d.UpdatedTime.Unix(),
		IsLiked:      isLiked,
		LikeCount:    uint64(d.LikeCount),
	}
}

func newArticleListResponse(models []*domaincontent.Article, total, lastID, page, pageSize uint64) *articledto.ArticleListResponse {
	resp := &articledto.ArticleListResponse{
		List:     make([]*articledto.ArticleListItem, 0, len(models)),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: pageSize,
	}
	for _, m := range models {
		summary := m.Content
		contentRune := []rune(m.Content)
		if len(contentRune) > 50 {
			summary = string(contentRune[:50]) + "..."
		}
		resp.List = append(resp.List, &articledto.ArticleListItem{
			ID:           m.ID,
			Title:        m.Title,
			Summary:      summary,
			AuthorID:     m.AuthorID,
			UpdatedTime:  m.UpdatedTime.Unix(),
			CommentCount: m.CommentCount,
			ViewCount:    m.ViewCount,
			LikeCount:    m.LikeCount,
		})
	}
	return resp
}

func newAdminListResponse(models []*domaincontent.Article, total, lastID, page, pageSize uint64) *articledto.AdminListResponse {
	resp := &articledto.AdminListResponse{
		List:     make([]*articledto.AdminListItem, 0, len(models)),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: pageSize,
	}
	for _, m := range models {
		resp.List = append(resp.List, &articledto.AdminListItem{
			ID:          m.ID,
			Title:       m.Title,
			Tags:        splitTags(m.Tags),
			Status:      m.Status,
			CreatedTime: m.CreatedTime.Unix(),
			UpdatedTime: m.UpdatedTime.Unix(),
		})
	}
	return resp
}

func newExternalListResponse(models []*domaincontent.Article, total, page, pageSize uint64) *articledto.ExternalListResponse {
	resp := &articledto.ExternalListResponse{
		List:     make([]*articledto.ExternalListItem, 0, len(models)),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
	for _, m := range models {
		resp.List = append(resp.List, &articledto.ExternalListItem{
			ID:          m.ID,
			Title:       m.Title,
			Tags:        splitTags(m.Tags),
			CreatedTime: m.CreatedTime.Unix(),
			UpdatedTime: m.UpdatedTime.Unix(),
		})
	}
	return resp
}

func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}
	return strings.Split(tags, ",")
}
