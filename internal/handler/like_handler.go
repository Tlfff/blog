package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/like"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
)

type LikeHandler struct {
	likeService *service.LikeService
}

func NewLikeHandler(likeService *service.LikeService) *LikeHandler {
	return &LikeHandler{
		likeService: likeService,
	}
}

// 用户点赞文章接口 -> POST /auth/article/like
func (h *LikeHandler) ArticleLikeHandler(c *gin.Context) {
	// 1. 解析请求的文章 ID
	var req like.ArticleIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
		return
	}

	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.likeService.ArticleLike(c, userCtx.UserID, req.ArticleID); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "点赞成功", nil)
}

// 取消文章点赞接口 -> POST /auth/article/cancel_like
func (h *LikeHandler) ArticleCancelLikeHandler(c *gin.Context) {
	// 1. 解析请求的文章 ID
	var req like.ArticleIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
		return
	}

	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.likeService.ArticleCancelLike(c, userCtx.UserID, req.ArticleID); err != nil {
		c.Error(err)
	}

	common.OK(c, "取消点赞成功", nil)
}

// CommentLikeHandler 用户点赞评论接口 -> POST /auth/comment/like
func (h *LikeHandler) CommentLikeHandler(c *gin.Context) {
	// 1. 解析请求的评论 ID（使用通用的 ArticleIdRequest 或新定义，这里遵循你的入参规范）
	var req like.CommentIdRequest // 或者是 comment.CommentIdRequest，依你定义的 struct 为准
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrCommentNotFound) // 对齐你的统一错误规范
		return
	}

	// 2. 从 Gin 上下文获取当前登录用户
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 调用 100% 走 Redis 的 Service 接口（传 req.ArticleID 代表传过来的评论ID字段）
	if err := h.likeService.CommentLike(c, userCtx.UserID, req.CommentID); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "点赞评论成功", nil)
}

// CommentCancelLikeHandler 用户取消点赞评论接口 -> POST /auth/comment/cancel_like
func (h *LikeHandler) CommentCancelLikeHandler(c *gin.Context) {
	var req like.CommentIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrCommentNotFound)
		return
	}

	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	if err := h.likeService.CommentCancelLike(c, userCtx.UserID, req.CommentID); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "取消点赞评论成功", nil)
}
