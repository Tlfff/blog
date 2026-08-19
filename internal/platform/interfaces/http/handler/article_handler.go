package handler

import (
	"blog/internal/article/application/dto"
	arcticleDto "blog/internal/article/application/dto"
	"blog/internal/article/domain"
	"blog/internal/platform/auth"
	"blog/internal/shared/common"
	"context"

	"github.com/gin-gonic/gin"
)

// ArticleUsecase 是文章生命周期、查询和图片能力的应用用例接口。
type ArticleUsecase interface {
	CreateArticle(ctx context.Context, authorID uint64, title, content string, tags []string, status int8) error
	UpdateArticle(ctx context.Context, articleID, authorID uint64, title, content string, tags []string, status int8) error
	DeleteArticle(ctx context.Context, articleID, userID uint64) error
	ClearArticle(ctx context.Context, articleID, userID uint64) error
	PublishArticle(ctx context.Context, articleID, userID uint64) error
	RecoverArticle(ctx context.Context, articleID, userID uint64) error
	GetPublishedArticle(ctx context.Context, articleID, userID uint64) (*article.ArticleDetailResponse, error)
	GetArticle(ctx context.Context, articleID, userID uint64) (*article.ArticleDetailResponse, error)
	GetPublishedList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool) (*article.ArticleListResponse, error)
	GetAdminList(ctx context.Context, page, pageSize, lastID uint64, isDesc bool, status int8) (*article.AdminListResponse, error)
	GetAvailableList(ctx context.Context, page, pageSize uint64, isDesc bool) (*article.ExternalListResponse, error)
	GetUploadURL(ctx context.Context, fileExt string) (uploadURL, url string, err error)
}

type ArticleHandler struct {
	article     ArticleUsecase
	articleRank ArticleRankUsecase
}

// ArticleRankUsecase 是文章热榜查询接口。
type ArticleRankUsecase interface {
	GetHotRank(ctx context.Context) (*article.HotRankResponse, error)
}

func NewArticleHandler(article ArticleUsecase, articleRank ArticleRankUsecase) *ArticleHandler {
	return &ArticleHandler{
		article:     article,
		articleRank: articleRank,
	}
}

// 获取文章图片上传凭证
func (h *ArticleHandler) GetImageUploadURL(c *gin.Context) {
	// 1. 解析请求体并放进req
	var req arcticleDto.GetImageUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}
	// 2. 获取凭证
	uploadURL, url, err := h.article.GetUploadURL(c, req.FileExt)
	if err != nil {
		c.Error(err)
		return
	}
	// 3. 返回凭证
	common.OK(c, "获取成功", gin.H{
		"upload_url": uploadURL,
		"url":        url,
	})
}

// 创建文章
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	var req arcticleDto.CreateArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
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
	common.OK(c, "文章创建成功", nil)
}

// 更新文章
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	var req arcticleDto.UpdateArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
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

	common.OK(c, "文章更新成功", nil)
}

// 删除文章(移去垃圾箱)
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	var req arcticleDto.DeleteArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}
	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.article.DeleteArticle(c.Request.Context(), req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "文章删除成功", nil)
}

// 发表文章
func (h *ArticleHandler) PublishArticle(c *gin.Context) {
	var req arcticleDto.PublishArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.article.PublishArticle(c, req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "文章发表成功", nil)
}

// 公开：查看文章详情
func (h *ArticleHandler) GetArticleDetail(c *gin.Context) {
	var req article.GetDetailRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
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
	common.OK(c, "查询成功", res)
}

// 管理者：查看文章详情
func (h *ArticleHandler) GetArticleDetailForMe(c *gin.Context) {
	var req article.GetDetailRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
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
	common.OK(c, "查询成功", res)
}

// 获取用户已发表文章列表
func (h *ArticleHandler) GetPublishedList(c *gin.Context) {
	var req article.GetPublishListRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrParameter)
		return
	}

	resList, err := h.article.GetPublishedList(c, req.Page, req.PageSize, req.LastID, req.IsDesc)
	if err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "获取发表列表成功", resList)
}

// 管理者：获取文章列表
func (h *ArticleHandler) GetAdminList(c *gin.Context) {
	var req article.GetAdminListRequest
	// 1. 校验参数
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrArticleStatusError)
		return
	}
	// // 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	// user := c.MustGet("currentUser").(*auth.UserContext)

	resList, err := h.article.GetAdminList(c, req.Page, req.PageSize, req.LastID, req.IsDesc, req.Status)
	if err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "获取文章列表成功", resList)
}

// 管理者：获取垃圾箱列表，不需要传状态，因为固定为0
func (h *ArticleHandler) GetTrashList(c *gin.Context) {
	var req article.GetAdminListRequest
	// 1. 校验参数
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrArticleStatusError)
		return
	}
	// 2.获取已删除文章列表
	resList, err := h.article.GetAdminList(c, req.Page, req.PageSize, req.LastID, req.IsDesc, domain.StatusDeleted)
	if err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "获取垃圾箱列表成功", resList)
}

// 管理者：恢复垃圾箱中的文章
func (h *ArticleHandler) RecoverArticle(c *gin.Context) {
	var req article.RecoverArticleRequest

	// 1. 获取文章id
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrArticleStatusError)
		return
	}
	user := c.MustGet("currentUser").(*auth.UserContext)
	// 2. 恢复文章
	if err := h.article.RecoverArticle(c, req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "恢复文章成功", nil)
}

// 管理者：硬删除垃圾箱中的文章
func (h *ArticleHandler) ClearArticle(c *gin.Context) {
	var req article.RecoverArticleRequest

	// 1. 获取文章id
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrArticleStatusError)
		return
	}
	user := c.MustGet("currentUser").(*auth.UserContext)
	// 2. 删除文章
	if err := h.article.ClearArticle(c, req.ID, uint64(user.UserID)); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "删除文章成功", nil)
}

// 获取文章排行榜
func (h *ArticleHandler) GetHotArticleRank(c *gin.Context) {
	// 1. 获取文章排行榜
	rankList, err := h.articleRank.GetHotRank(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "获取排行榜成功", rankList)
}
