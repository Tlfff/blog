package http

import (
	articleapp "blog/internal/article/app"
	articleresult "blog/internal/article/app/dto"
	"blog/internal/article/domain"
	"blog/internal/platform/interfaces/http/response"
	auth "blog/internal/platform/security"
	apperrors "blog/internal/shared/apperrors"
	"context"

	"github.com/gin-gonic/gin"
)

// ArticleUsecase 是文章生命周期、查询和图片能力的应用用例接口。
type ArticleUsecase interface {
	// CreateArticle 创建文章。
	CreateArticle(ctx context.Context, authorID uint64, title, content string, tags []string, status int8) error
	// UpdateArticle 更新文章。
	UpdateArticle(ctx context.Context, articleID, authorID uint64, title, content string, tags []string, status int8) error
	// DeleteArticle 将文章移入垃圾箱。
	DeleteArticle(ctx context.Context, articleID, userID uint64) error
	// ClearArticle 彻底删除垃圾箱文章。
	ClearArticle(ctx context.Context, articleID, userID uint64) error
	// PublishArticle 发布文章。
	PublishArticle(ctx context.Context, articleID, userID uint64) error
	// RecoverArticle 恢复垃圾箱文章。
	RecoverArticle(ctx context.Context, articleID, userID uint64) error
	// GetPublishedArticle 查询公开文章详情。
	GetPublishedArticle(ctx context.Context, articleID, userID uint64) (*articleresult.ArticleDetailResponse, error)
	// GetArticle 查询作者文章详情。
	GetArticle(ctx context.Context, articleID, userID uint64) (*articleresult.ArticleDetailResponse, error)
	// GetPublishedList 查询公开文章列表。
	GetPublishedList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool) (*articleresult.ArticleListResponse, error)
	// GetAdminList 查询后台文章列表。
	GetAdminList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool, status int8) (*articleresult.AdminListResponse, error)
	// GetAvailableList 查询对外开放的文章列表。
	GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*articleresult.ExternalListResponse, error)
	// GetImageUploadURL 获取单张文章图片上传凭证。
	GetImageUploadURL(ctx context.Context, command articleapp.GetImageUploadURLCommand) (*articleresult.ImageUploadCredentialResponse, error)
}

// ArticleHandler 处理 Article 上下文的 HTTP 请求。
type ArticleHandler struct {
	article     ArticleUsecase     // 文章应用用例
	articleRank ArticleRankUsecase // 文章热榜应用用例
}

// ArticleRankUsecase 是文章热榜查询接口。
type ArticleRankUsecase interface {
	// GetHotRank 查询文章热榜。
	GetHotRank(ctx context.Context) (*articleresult.HotRankResponse, error)
}

// NewArticleHandler 创建 Article HTTP Handler。
func NewArticleHandler(article ArticleUsecase, articleRank ArticleRankUsecase) *ArticleHandler {
	return &ArticleHandler{
		article:     article,
		articleRank: articleRank,
	}
}

// GetImageUploadURL 获取单张文章图片上传凭证。
func (h *ArticleHandler) GetImageUploadURL(c *gin.Context) {
	// 1. 解析并校验单图片上传凭证请求
	var req GetImageUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	// 2. 获取上传凭证并返回图片标识和实时预览地址
	result, err := h.article.GetImageUploadURL(c, articleapp.GetImageUploadURLCommand{FileExt: req.FileExt})
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "获取成功", result)
}

// CreateArticle 创建文章。
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	var req CreateArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)
	// 3. 调用service创建文章
	err := h.article.CreateArticle(
		c,
		uint64(user.UserID),
		req.Title,
		req.Content,
		req.Tags,
		req.Status,
	)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "文章创建成功", nil)
}

// UpdateArticle 更新文章。
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	var req UpdateArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}
	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)

	err := h.article.UpdateArticle(
		c,
		req.ID,
		uint64(user.UserID),
		req.Title,
		req.Content,
		req.Tags,
		req.Status,
	)

	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "文章更新成功", nil)
}

// DeleteArticle 将文章移入垃圾箱。
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	var req DeleteArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}
	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.article.DeleteArticle(c.Request.Context(), req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "文章删除成功", nil)
}

// PublishArticle 发布文章。
func (h *ArticleHandler) PublishArticle(c *gin.Context) {
	var req PublishArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody)
		return
	}

	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.article.PublishArticle(c, req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "文章发表成功", nil)
}

// GetArticleDetail 查询公开文章详情。
func (h *ArticleHandler) GetArticleDetail(c *gin.Context) {
	var req GetDetailRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrArticleIDInvalid)
		return
	}

	// 2. 从Gin 上下文获取用户信息，可能未登录
	userID := uint64(0)
	if userCtx, exists := c.Get("currentUser"); exists {
		userID = userCtx.(*auth.UserContext).UserID
	}
	// 3. 获取详情
	res, err := h.article.GetPublishedArticle(c, req.ID, userID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "查询成功", res)
}

// GetArticleDetailForMe 查询管理员文章详情。
func (h *ArticleHandler) GetArticleDetailForMe(c *gin.Context) {
	var req GetDetailRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrArticleIDInvalid)
		return
	}

	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	// 3. 获取详情
	res, err := h.article.GetArticle(c, req.ID, userCtx.UserID)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "查询成功", res)
}

// GetPublishedList 查询已发表文章列表。
func (h *ArticleHandler) GetPublishedList(c *gin.Context) {
	var req GetPublishListRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrParameter)
		return
	}

	resList, err := h.article.GetPublishedList(c, req.Page, req.PageSize, req.LastID, req.IsDesc)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "获取发表列表成功", resList)
}

// GetAdminList 查询后台文章列表。
func (h *ArticleHandler) GetAdminList(c *gin.Context) {
	var req GetAdminListRequest
	// 1. 校验参数
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrArticleStatusError)
		return
	}
	// // 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	// user := c.MustGet("currentUser").(*auth.UserContext)

	resList, err := h.article.GetAdminList(c, req.Page, req.PageSize, req.LastID, req.IsDesc, req.Status)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "获取文章列表成功", resList)
}

// GetTrashList 查询垃圾箱文章列表。
func (h *ArticleHandler) GetTrashList(c *gin.Context) {
	var req GetAdminListRequest
	// 1. 校验参数
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrArticleStatusError)
		return
	}
	// 2.获取已删除文章列表
	resList, err := h.article.GetAdminList(c, req.Page, req.PageSize, req.LastID, req.IsDesc, domain.StatusDeleted.Int8())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "获取垃圾箱列表成功", resList)
}

// RecoverArticle 恢复垃圾箱中的文章。
func (h *ArticleHandler) RecoverArticle(c *gin.Context) {
	var req RecoverArticleRequest

	// 1. 获取文章id
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrArticleStatusError)
		return
	}
	user := c.MustGet("currentUser").(*auth.UserContext)
	// 2. 恢复文章
	if err := h.article.RecoverArticle(c, req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "恢复文章成功", nil)
}

// ClearArticle 彻底删除垃圾箱中的文章。
func (h *ArticleHandler) ClearArticle(c *gin.Context) {
	var req RecoverArticleRequest

	// 1. 获取文章id
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrArticleStatusError)
		return
	}
	user := c.MustGet("currentUser").(*auth.UserContext)
	// 2. 删除文章
	if err := h.article.ClearArticle(c, req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "删除文章成功", nil)
}

// GetHotArticleRank 查询文章热榜。
func (h *ArticleHandler) GetHotArticleRank(c *gin.Context) {
	// 1. 获取文章排行榜
	rankList, err := h.articleRank.GetHotRank(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, "获取排行榜成功", rankList)
}
