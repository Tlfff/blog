package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/article"
	arcticleDto "blog/internal/dto/article"
	"blog/internal/model"
	"blog/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type ArticleHandler struct {
	article *service.ArticleService
}

func NewArticleHandler(article *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{article: article}
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

	article := &model.Article{
		Title:      req.Title,
		Content:    req.Content,
		Tags:       req.Tags,
		Status:     req.Status,
		AuthorID:   user.UserID,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	}

	if err := h.article.CreateArticle(article); err != nil {
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

	article := &model.Article{
		ID:         req.ID,
		Title:      req.Title,
		Content:    req.Content,
		Tags:       req.Tags,
		Status:     req.Status,
		AuthorID:   user.UserID,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	}

	if err := h.article.UpdateArticle(article); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "文章更新成功", nil)
}

// 删除文章
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	var req arcticleDto.DeleteArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}
	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.article.DeleteArticle(req.ID, user.UserID); err != nil {
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

	if err := h.article.PublishArticle(req.ID, user.UserID); err != nil {
		c.Error(err)
		return
	}
	common.OK(c, "文章发表成功", nil)
}

// 获取文章详情
// 场景A（公开接口）：仅允许查看已发表的文章
func (h *ArticleHandler) GetArticleDetail(c *gin.Context) {
	var req article.GetDetailRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
		return
	}

	// 2. 获取详情
	article, err := h.article.GetArticle(req.ID)
	if err != nil {
		c.Error(err)
		return
	}
	// 3. 只能看见已发表的文章
	if article.Status != model.Published {
		c.Error(common.ErrArticleNotFound)
		return
	}
	res := arcticleDto.NewArticleDetailResponse(article)
	common.OK(c, "查询成功", res)
}

// 场景 B（需要登录）：创作者看自己未发表的文章，如草稿
func (h *ArticleHandler) GetArticleDetailForMe(c *gin.Context) {
	var req article.GetDetailRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
		return
	}

	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 4. 获取详情
	articleData, err := h.article.GetArticle(req.ID)
	if err != nil {
		c.Error(err)
		return
	}

	// 5. 判断自己是否是作者
	if articleData.AuthorID != userCtx.UserID {
		c.Error(common.ErrArticlePermissionDenied)
		return
	}

	res := article.NewArticleDetailResponse(articleData)
	common.OK(c, "查询成功", res)
}

// 获取用户已发表文章列表
func (h *ArticleHandler) GetPublishedList(c *gin.Context) {
	var req article.GetPublishListRequest
	// 1. 自动去 Query 拿 ?id=xxx，自动转成 int64，自动校验 min=1
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrUserNotFound)
		return
	}

	articleList, err := h.article.GetPublishedList(req.AuthorID)
	if err != nil {
		c.Error(err)
		return
	}
	resList := arcticleDto.NewArticleListResponse(articleList)
	common.OK(c, "获取发表列表成功", resList)
}

// 获取用户草稿文章列表
func (h *ArticleHandler) GetDraftedList(c *gin.Context) {
	// 1. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)

	articleList, err := h.article.GetDraftedList(user.UserID)
	if err != nil {

		c.Error(err)
		return
	}

	resList := arcticleDto.NewArticleListResponse(articleList)
	common.OK(c, "获取草稿列表成功", resList)
}
