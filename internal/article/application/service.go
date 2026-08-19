// Package application 编排 Article 上下文应用用例。
package application

import (
	articledto "blog/internal/article/application/dto"
	domaincontent "blog/internal/article/domain"
	"blog/internal/shared/common"
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

// 文章正文中的图片 URL 匹配规则，用于把临时图片转正
var articleImageURLRe = regexp.MustCompile(`https?://[^\s"'<>()]+`)

// Service 编排文章生命周期、查询与图片用例。
type Service struct {
	repo         domaincontent.ArticleRepository       // 文章持久化 Port
	users        domaincontent.UserQuery               // 作者信息查询 Port，由 Identity 侧提供
	images       domaincontent.ArticleImageStorage     // 文章图片对象存储 Port
	interactions domaincontent.ArticleInteractionQuery // 文章互动统计查询 Port，由 Community 侧提供
	publicDomain string                                // 对象存储对外访问域名
	allowedExts  map[string]bool                       // 允许上传的图片扩展名集合
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
	// 1. 把允许的扩展名统一转小写后放入集合，便于后续校验
	extMap := make(map[string]bool, len(allowedExts))
	for _, ext := range allowedExts {
		extMap[strings.ToLower(ext)] = true
	}
	// 2. 组装并返回应用服务实例
	return &Service{
		repo:         repo,
		users:        users,
		images:       images,
		interactions: interactions,
		publicDomain: publicDomain,
		allowedExts:  extMap,
	}
}

// 创建文章，并把正文里的临时图片转正到文章正式目录
func (s *Service) CreateArticle(ctx context.Context, authorID uint64, title, content string, tags []string, status int8) error {
	// 1. 组装文章聚合根，标签以英文逗号拼接存储
	art := &domaincontent.Article{
		AuthorID: authorID,
		Title:    title,
		Content:  content,
		Tags:     strings.Join(tags, ","),
		Status:   status,
	}
	// 2. 先落库拿到文章ID，图片转正需要用它作为目录名
	if err := s.repo.Create(ctx, art); err != nil {
		return err
	}

	// 3. 把正文中的临时图片迁移到文章正式目录，失败则清除刚创建的文章
	newContent, err := s.PromoteImages(ctx, art.ID, content)
	if err != nil {
		_ = s.repo.Clear(ctx, art.ID)
		return err
	}
	// 4. 正文有变化时回写数据库
	if newContent != content {
		art.Content = newContent
		return s.repo.Update(ctx, art)
	}
	return nil
}

// 更新文章内容，仅作者本人可编辑且已删除文章不可编辑
func (s *Service) UpdateArticle(ctx context.Context, articleID, authorID uint64, title, content string, tags []string, status int8) error {
	// 1. 查询原文章并校验删除状态与编辑权限
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

	// 2. 把正文中新增的临时图片转正
	content, err = s.PromoteImages(ctx, articleID, content)
	if err != nil {
		return err
	}

	// 3. 覆盖更新文章字段
	art := &domaincontent.Article{
		ID:      articleID,
		Title:   title,
		Content: content,
		Tags:    strings.Join(tags, ","),
		Status:  status,
	}
	return s.repo.Update(ctx, art)
}

// 软删除文章，仅作者本人可删除
func (s *Service) DeleteArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询原文章并校验删除权限
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if !oldArticle.CanDelete(userID) {
		return common.ErrArticlePermissionDenied
	}
	// 2. 执行软删除，仅改状态不删数据
	return s.repo.SoftDelete(ctx, articleID)
}

// 物理清除文章，仅作者本人可操作
func (s *Service) ClearArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询原文章并校验权限
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if !oldArticle.CanDelete(userID) {
		return common.ErrArticlePermissionDenied
	}
	// 2. 物理删除文章记录
	return s.repo.Clear(ctx, articleID)
}

// 发表文章，仅作者本人可发表且已删除文章不可发表
func (s *Service) PublishArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询原文章并校验发表权限与删除状态
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
	// 2. 更新状态为已发表
	oldArticle.Publish()
	return s.repo.Update(ctx, oldArticle)
}

// 将已删除文章恢复为草稿状态
func (s *Service) RecoverArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询原文章
	oldArticle, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	// 2. 恢复为草稿并回写作者
	oldArticle.Recover()
	oldArticle.AuthorID = userID
	return s.repo.Update(ctx, oldArticle)
}

// 获取已发表文章详情，供游客与普通用户访问
func (s *Service) GetPublishedArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	// 1. 查询文章详情
	detail, err := s.findDetail(ctx, articleID)
	if err != nil {
		return nil, err
	}
	// 2. 校验删除状态与公开可见性
	if detail.IsDeleted() {
		return nil, common.ErrArticleDeleted
	}
	if !detail.IsPubliclyVisible() {
		return nil, common.ErrArticlePermissionDenied
	}
	// 3. 组装详情响应
	return s.buildDetail(ctx, detail, userID)
}

// 获取文章详情，允许作者本人查看草稿
func (s *Service) GetArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	// 1. 查询文章详情
	detail, err := s.findDetail(ctx, articleID)
	if err != nil {
		return nil, err
	}
	// 2. 已删除文章不可访问
	if detail.IsDeleted() {
		return nil, common.ErrArticleDeleted
	}
	// 3. 组装详情响应
	return s.buildDetail(ctx, detail, userID)
}

// 分页获取已发表文章列表，支持游标与偏移两种分页
func (s *Service) GetPublishedList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool) (*articledto.ArticleListResponse, error) {
	var list []*domaincontent.Article
	var err error
	// 1. 有游标走游标分页，否则走偏移分页
	if lastID > 0 {
		list, err = s.repo.ListWithCursor(ctx, lastID, int(pageSize), isDesc, domaincontent.StatusPublished)
	} else {
		list, err = s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, domaincontent.StatusPublished)
	}
	if err != nil {
		return nil, err
	}
	// 2. 统计已发表文章总数
	total, err := s.repo.CountByStatus(ctx, domaincontent.StatusPublished)
	if err != nil {
		return nil, err
	}
	// 3. 计算下一页游标锚点
	nextLastID := uint64(0)
	if len(list) > 0 {
		nextLastID = list[len(list)-1].ID
	}
	// 4. 组装列表响应
	return newArticleListResponse(list, uint64(total), nextLastID, page, pageSize), nil
}

// 分页获取后台文章列表，可按状态过滤
func (s *Service) GetAdminList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool, status int8) (*articledto.AdminListResponse, error) {
	var list []*domaincontent.Article
	var err error
	// 1. 有游标走游标分页，否则走偏移分页
	if lastID > 0 {
		list, err = s.repo.ListWithCursor(ctx, lastID, int(pageSize), isDesc, status)
	} else {
		list, err = s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, status)
	}
	if err != nil {
		return nil, err
	}
	// 2. 按状态统计总数
	total, err := s.repo.CountByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	// 3. 计算下一页游标锚点
	nextLastID := uint64(0)
	if len(list) > 0 {
		nextLastID = list[len(list)-1].ID
	}
	return newAdminListResponse(list, uint64(total), nextLastID, page, pageSize), nil
}

// 分页获取对外开放的已发表文章列表，供二方服务调用
func (s *Service) GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*articledto.ExternalListResponse, error) {
	// 1. 仅按偏移分页查询已发表文章
	list, err := s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, domaincontent.StatusPublished)
	if err != nil {
		return nil, err
	}
	// 2. 统计已发表文章总数
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

// 生成文章图片的预签名上传地址与访问地址
func (s *Service) GetUploadURL(ctx context.Context, fileExt string) (uploadURL, url string, err error) {
	// 1. 校验对象存储是否可用
	if s.images == nil {
		return "", "", common.ErrSystem
	}
	// 2. 校验扩展名是否在白名单内
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	if !s.allowedExts[ext] {
		return "", "", common.ErrInvalidRequestBody
	}
	// 3. 生成临时目录下的对象名并签发上传地址
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

// 组装文章详情响应，补齐作者信息与当前用户点赞状态
func (s *Service) buildDetail(ctx context.Context, detail *domaincontent.ArticleWithAuthor, userID uint64) (*articledto.ArticleDetailResponse, error) {
	// 1. 补齐作者昵称、头像与最后登录IP
	if s.users != nil {
		if user, err := s.users.FindUserByID(ctx, detail.AuthorID); err == nil {
			detail.Nickname = user.Nickname
			detail.Avatar = user.Avatar
			detail.LastLoginIP = user.LastLoginIP
		}
	}
	// 2. 登录用户才查询点赞状态
	isLiked := false
	if userID > 0 && s.interactions != nil {
		isLiked, _ = s.interactions.IsUserLikedArticle(ctx, userID, detail.ID)
	}
	// 3. 转换为详情响应 DTO
	return newArticleDetailResponse(detail, isLiked), nil
}

// 按ID查询文章，并把领域错误映射为统一业务错误
func (s *Service) findArticle(ctx context.Context, articleID uint64) (*domaincontent.Article, error) {
	article, err := s.repo.FindByID(ctx, articleID)
	if errors.Is(err, domaincontent.ErrArticleNotFound) {
		return nil, common.ErrArticleNotFound
	}
	return article, err
}

// 按ID查询文章详情，并把领域错误映射为统一业务错误
func (s *Service) findDetail(ctx context.Context, articleID uint64) (*domaincontent.ArticleWithAuthor, error) {
	detail, err := s.repo.FindWithAuthorByID(ctx, articleID)
	if errors.Is(err, domaincontent.ErrArticleNotFound) {
		return nil, common.ErrArticleNotFound
	}
	return detail, err
}

// 把文章详情领域模型转换为详情响应 DTO
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

// 把文章列表领域模型转换为前台列表响应 DTO
func newArticleListResponse(models []*domaincontent.Article, total, lastID, page, pageSize uint64) *articledto.ArticleListResponse {
	resp := &articledto.ArticleListResponse{
		List:     make([]*articledto.ArticleListItem, 0, len(models)),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: pageSize,
	}
	for _, m := range models {
		// 1. 截取正文前50个字符作为摘要
		summary := m.Content
		contentRune := []rune(m.Content)
		if len(contentRune) > 50 {
			summary = string(contentRune[:50]) + "..."
		}
		// 2. 映射为列表项
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

// 把文章列表领域模型转换为后台列表响应 DTO
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

// 把文章列表领域模型转换为对外开放列表响应 DTO
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

// 拆分逗号分隔的标签字符串，空字符串返回空切片
func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}
	return strings.Split(tags, ",")
}
