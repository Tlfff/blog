// Package application 编排 Article 上下文应用用例。
package app

import (
	articledto "blog/internal/article/app/dto"
	domaincontent "blog/internal/article/domain"
	iputil "blog/internal/platform/ip"
	apperrors "blog/internal/shared/apperrors"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

const articleImageUploadURLTTL = 10 * time.Minute

// Service 编排文章生命周期、查询与图片用例。
type Service struct {
	repo         domaincontent.ArticleRepository       // 文章持久化 Port
	users        domaincontent.UserQuery               // 作者信息查询 Port，由 Identity 侧提供
	images       domaincontent.ArticleImageStorage     // 文章图片对象存储 Port
	interactions domaincontent.ArticleInteractionQuery // 文章互动统计查询 Port，由 Community 侧提供
	publicDomain string                                // 对象存储对外访问域名
	allowedExts  map[string]bool                       // 允许上传的图片扩展名集合
}

// NewService 创建 Article Application 服务。
//
// 参数说明：
//   - repo：文章持久化 Port。
//   - users：作者公开信息查询 Port。
//   - images：文章图片对象存储 Port。
//   - interactions：文章互动状态查询 Port。
//   - publicDomain：对象存储公开访问域名。
//   - allowedExts：允许上传的图片扩展名列表。
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

// InitializeArticle 初始化空内容文章草稿并返回文章唯一标识。
func (s *Service) InitializeArticle(ctx context.Context, authorID uint64) (*articledto.InitializeArticleResponse, error) {
	// 1. 创建仅用于取得文章 ID 的初始化草稿
	article, err := domaincontent.NewDraftArticle(authorID)
	if err != nil {
		return nil, mapArticleError(err)
	}

	// 2. 保存草稿并返回 Repository 回填的文章 ID
	if err := s.repo.Create(ctx, article); err != nil {
		return nil, err
	}
	return &articledto.InitializeArticleResponse{ArticleID: article.ID}, nil
}

// CreateArticle 创建文章。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - authorID：作者用户唯一标识。
//   - title：文章标题。
//   - content：文章正文。
//   - tags：文章标签列表。
//   - status：文章状态：0-未指定；1-已删除；2-草稿；3-已发表。
func (s *Service) CreateArticle(ctx context.Context, authorID uint64, title, content string, tags []string, status int8) error {
	// 1. 通过领域构造函数创建文章聚合，标签仍以英文逗号拼接存储
	art, err := domaincontent.NewArticle(authorID, title, content, strings.Join(tags, ","), status)
	if err != nil {
		return mapArticleError(err)
	}
	// 2. 保存已经包含正式图片 URL 的文章正文
	return s.repo.Create(ctx, art)
}

// UpdateArticle 更新文章内容，仅作者本人可编辑且已删除文章不可编辑。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - articleID：文章唯一标识。
//   - authorID：当前作者用户唯一标识。
//   - title：新文章标题。
//   - content：新文章正文。
//   - tags：新文章标签列表。
//   - status：新文章状态：0-未指定；1-已删除；2-草稿；3-已发表。
func (s *Service) UpdateArticle(ctx context.Context, articleID, authorID uint64, title, content string, tags []string, status int8) error {
	// 1. 查询文章并完成领域校验和内存状态更新
	article, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if err := article.EditBy(authorID, title, content, strings.Join(tags, ","), status); err != nil {
		return mapArticleError(err)
	}

	// 2. 保存已经包含正式图片 URL 的文章正文
	return s.repo.Update(ctx, article)
}

// DeleteArticle 将作者文章软删除并移入垃圾箱。
func (s *Service) DeleteArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询文章并执行移入垃圾箱领域行为
	article, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if err := article.MoveToTrashBy(userID); err != nil {
		return mapArticleError(err)
	}

	// 2. 保持现有 Repository 软删除调用
	return s.repo.SoftDelete(ctx, articleID)
}

// ClearArticle 彻底删除已进入垃圾箱的作者文章。
func (s *Service) ClearArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询文章并校验作者身份和垃圾箱状态
	article, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if err := article.EnsureCanPermanentlyDeleteBy(userID); err != nil {
		return mapArticleError(err)
	}

	// 2. 先清理文章正式目录，失败时保留数据库记录以便重试
	if s.images == nil {
		return apperrors.ErrSystem
	}
	if err := s.images.DeleteObjectsByPrefix(ctx, articleImageObjectPrefix(articleID)); err != nil {
		return err
	}

	// 3. 图片清理成功后物理删除文章记录
	return s.repo.Clear(ctx, articleID)
}

// PublishArticle 发表作者文章，已删除文章不可发表。
func (s *Service) PublishArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询文章并执行发布领域行为
	article, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if err := article.PublishBy(userID); err != nil {
		return mapArticleError(err)
	}

	// 2. 保存发布后的聚合
	return s.repo.Update(ctx, article)
}

// RecoverArticle 将已删除文章恢复为草稿状态。
func (s *Service) RecoverArticle(ctx context.Context, articleID, userID uint64) error {
	// 1. 查询文章并执行恢复领域行为
	article, err := s.findArticle(ctx, articleID)
	if err != nil {
		return err
	}
	if err := article.RecoverBy(userID); err != nil {
		return mapArticleError(err)
	}

	// 2. 保存恢复后的聚合，作者身份保持不变
	return s.repo.Update(ctx, article)
}

// GetPublishedArticle 获取已发表文章详情，供游客与普通用户访问。
func (s *Service) GetPublishedArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	// 1. 查询文章详情
	detail, err := s.findDetail(ctx, articleID)
	if err != nil {
		return nil, err
	}
	// 2. 校验删除状态与公开可见性
	if detail.IsDeleted() {
		return nil, apperrors.ErrArticleDeleted
	}
	if !detail.IsPubliclyVisible() {
		return nil, apperrors.ErrArticlePermissionDenied
	}
	// 3. 组装详情响应
	return s.buildDetail(ctx, detail, userID)
}

// GetArticle 获取文章详情，允许作者本人查看草稿。
func (s *Service) GetArticle(ctx context.Context, articleID, userID uint64) (*articledto.ArticleDetailResponse, error) {
	// 1. 查询文章详情
	detail, err := s.findDetail(ctx, articleID)
	if err != nil {
		return nil, err
	}
	// 2. 已删除文章不可访问
	if detail.IsDeleted() {
		return nil, apperrors.ErrArticleDeleted
	}
	// 3. 组装详情响应
	return s.buildDetail(ctx, detail, userID)
}

// GetPublishedList 分页获取已发表文章列表，支持游标与偏移两种分页。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - page：当前页码。
//   - pageSize：每页数量。
//   - lastID：游标文章唯一标识，为 0 时使用 Offset 分页。
//   - isDesc：是否按文章唯一标识倒序排列。
func (s *Service) GetPublishedList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool) (*articledto.ArticleListResponse, error) {
	var list []*domaincontent.Article
	var err error
	// 1. 有游标走游标分页，否则走偏移分页
	if lastID > 0 {
		list, err = s.repo.ListWithCursor(ctx, lastID, int(pageSize), isDesc, domaincontent.StatusPublished.Int8())
	} else {
		list, err = s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, domaincontent.StatusPublished.Int8())
	}
	if err != nil {
		return nil, err
	}
	// 2. 统计已发表文章总数
	total, err := s.repo.CountByStatus(ctx, domaincontent.StatusPublished.Int8())
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

// GetAdminList 分页获取后台文章列表，可按状态过滤。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - page：当前页码。
//   - pageSize：每页数量。
//   - lastID：游标文章唯一标识，为 0 时使用 Offset 分页。
//   - isDesc：是否按文章唯一标识倒序排列。
//   - status：文章状态过滤值。
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

// GetAvailableList 分页获取对外开放的已发表文章列表，供二方服务调用。
func (s *Service) GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*articledto.ExternalListResponse, error) {
	// 1. 仅按偏移分页查询已发表文章
	list, err := s.repo.ListWithOffset(ctx, int(page), int(pageSize), isDesc, domaincontent.StatusPublished.Int8())
	if err != nil {
		return nil, err
	}
	// 2. 统计已发表文章总数
	total, err := s.repo.CountByStatus(ctx, domaincontent.StatusPublished.Int8())
	if err != nil {
		return nil, err
	}
	return newExternalListResponse(list, uint64(total), page, pageSize), nil
}

// GetArticleInfo 返回文章供服务间只读查询使用的基本信息。
func (s *Service) GetArticleInfo(ctx context.Context, articleID uint64) (*domaincontent.Article, error) {
	return s.findArticle(ctx, articleID)
}

// GetImageUploadURLs 批量生成直接写入文章正式目录的图片上传凭证。
func (s *Service) GetImageUploadURLs(ctx context.Context, command GetImageUploadURLsCommand) (*articledto.ImageUploadCredentialsResponse, error) {
	// 1. 校验对象存储与待上传文件列表
	if s.images == nil {
		return nil, apperrors.ErrSystem
	}
	if len(command.Files) == 0 {
		return nil, apperrors.ErrInvalidRequestBody
	}

	// 2. 加载文章并执行 作者及删除状态校验
	article, err := s.findArticle(ctx, command.ArticleID)
	if err != nil {
		return nil, err
	}
	if err := article.EnsureCanUploadImageBy(command.AuthorID); err != nil {
		return nil, mapArticleError(err)
	}

	// 3. 在签发任何凭证前整体校验客户端标识和扩展名
	extensions := make([]string, len(command.Files))
	for index, file := range command.Files {
		// 校验客户端标识是否为空
		if strings.TrimSpace(file.ClientID) == "" {
			return nil, apperrors.ErrInvalidRequestBody
		}
		// 检查扩展名是否在白名单内
		extension := strings.ToLower(strings.TrimPrefix(file.FileExt, "."))
		if !s.allowedExts[extension] {
			return nil, apperrors.ErrInvalidRequestBody
		}
		extensions[index] = extension
	}

	// 4. 为每张图片生成正式对象 Key 和预签名上传地址
	credentials := make([]articledto.ImageUploadCredential, 0, len(command.Files))
	for index, file := range command.Files {
		objectKey := path.Join(articleImageObjectPrefix(command.ArticleID), uuid.NewString()+"."+extensions[index])
		uploadURL, err := s.images.PresignedPutURL(ctx, objectKey, articleImageUploadURLTTL)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, articledto.ImageUploadCredential{
			ClientID:  file.ClientID,
			UploadURL: uploadURL,
			URL:       s.images.GetObjectURL(s.publicDomain, objectKey),
		})
	}

	// 5. 返回与请求顺序一致的批量上传凭证
	return &articledto.ImageUploadCredentialsResponse{Files: credentials}, nil
}

// articleImageObjectPrefix 返回文章图片正式目录前缀。
func articleImageObjectPrefix(articleID uint64) string {
	// 1. 使用尾部斜杠限制前缀只匹配目标文章目录
	return path.Join("article", fmt.Sprint(articleID)) + "/"
}

// 组装文章详情响应，补齐作者信息与当前用户点赞状态
func (s *Service) buildDetail(ctx context.Context, detail *domaincontent.ArticleWithAuthor, userID uint64) (*articledto.ArticleDetailResponse, error) {
	// 1. 只读 JOIN 未返回作者时，通过 User Application Facade 补齐最小快照
	if s.users != nil && detail.Nickname == "" {
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

// mapArticleError 将领域错误映射为现有对外应用错误。
func mapArticleError(err error) error {
	switch {
	case errors.Is(err, domaincontent.ErrArticleNotFound):
		return apperrors.ErrArticleNotFound
	case errors.Is(err, domaincontent.ErrArticleDeleted):
		return apperrors.ErrArticleDeleted
	case errors.Is(err, domaincontent.ErrArticlePermissionDenied):
		return apperrors.ErrArticlePermissionDenied
	case errors.Is(err, domaincontent.ErrArticleStatusError):
		return apperrors.ErrArticleStatusError
	case errors.Is(err, domaincontent.ErrArticleTitleEmpty):
		return apperrors.ErrArticleTitleEmpty
	case errors.Is(err, domaincontent.ErrArticleContentEmpty):
		return apperrors.ErrArticleContentEmpty
	case errors.Is(err, domaincontent.ErrArticleStatusInvalid):
		return apperrors.ErrArticleStatusInvalid
	default:
		return err
	}
}

// 按ID查询文章，并把领域错误映射为统一业务错误
func (s *Service) findArticle(ctx context.Context, articleID uint64) (*domaincontent.Article, error) {
	article, err := s.repo.FindByID(ctx, articleID)
	if err != nil {
		return nil, mapArticleError(err)
	}
	return article, nil
}

// 按ID查询文章详情，并把领域错误映射为统一业务错误
func (s *Service) findDetail(ctx context.Context, articleID uint64) (*domaincontent.ArticleWithAuthor, error) {
	detail, err := s.repo.FindWithAuthorByID(ctx, articleID)
	if err != nil {
		return nil, mapArticleError(err)
	}
	return detail, nil
}

// 把文章详情领域模型转换为详情响应 DTO
func newArticleDetailResponse(d *domaincontent.ArticleWithAuthor, isLiked bool) *articledto.ArticleDetailResponse {
	tags := splitTags(d.Tags)
	return &articledto.ArticleDetailResponse{
		ID:           d.ID,
		Title:        d.Title,
		Content:      d.Content,
		Tags:         tags,
		Status:       d.Status.Int8(),
		AuthorNick:   d.Nickname,
		AuthorAvatar: d.Avatar,
		IP:           iputil.ConvertIPToRegion(d.LastLoginIP),
		CreatedTime:  d.CreatedTime.Unix(),
		UpdatedTime:  d.UpdatedTime.Unix(),
		IsLiked:      isLiked,
		LikeCount:    uint64(d.LikeCount),
	}
}

// newArticleListResponse 把文章列表领域模型转换为前台列表响应 DTO。
//
// 参数说明：
//   - models：文章领域对象列表。
//   - total：符合条件的文章总数。
//   - lastID：下一页查询使用的游标文章 ID。
//   - page：当前页码。
//   - pageSize：每页数量。
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

// newAdminListResponse 把文章列表领域模型转换为后台列表响应 DTO。
//
// 参数说明：
//   - models：文章领域对象列表。
//   - total：符合条件的文章总数。
//   - lastID：下一页查询使用的游标文章 ID。
//   - page：当前页码。
//   - pageSize：每页数量。
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
			Status:      m.Status.Int8(),
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

// GetArticleSnapshot 返回跨上下文查询所需的最小文章快照。
func (s *Service) GetArticleSnapshot(ctx context.Context, articleID uint64) (id, authorID uint64, title string, err error) {
	article, err := s.findArticle(ctx, articleID)
	if err != nil {
		return 0, 0, "", err
	}
	return article.ID, article.AuthorID, article.Title, nil
}
