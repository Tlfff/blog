package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/like"
	"context"

	"github.com/gin-gonic/gin"
)

// ArticleLikeUsecase 是文章点赞应用用例接口。
type ArticleLikeUsecase interface {
	ArticleLike(ctx context.Context, userID, articleID uint64) error
	ArticleCancelLike(ctx context.Context, userID, articleID uint64) error
	IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error)
}

// CommentLikeUsecase 是评论点赞应用用例接口。
type CommentLikeUsecase interface {
	CommentLike(ctx context.Context, userID, commentID uint64) error
	CommentCancelLike(ctx context.Context, userID, commentID uint64) error
}

type LikeHandler struct {
	artLikeService ArticleLikeUsecase
	comLikeService CommentLikeUsecase
}

func NewLikeHandler(artLikeService ArticleLikeUsecase, comLikeService CommentLikeUsecase) *LikeHandler {
	return &LikeHandler{
		artLikeService: artLikeService,
		comLikeService: comLikeService,
	}
}

// 用户点赞文章接口
func (h *LikeHandler) ArticleLikeHandler(c *gin.Context) {
	// 1. 解析请求的文章 ID
	var req like.ArticleIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
		return
	}

	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 点赞文章，直接操作redis
	if err := h.artLikeService.ArticleLike(c.Request.Context(), userCtx.UserID, req.ArticleID); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "点赞成功", nil)
}

// 用户取消文章点赞
func (h *LikeHandler) ArticleCancelLikeHandler(c *gin.Context) {
	// 1. 解析请求的文章 ID
	var req like.ArticleIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrArticleIDInvalid)
		return
	}

	// 2. 从Gin 上下文获取用户信息
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 取消点赞文章，直接操作redis
	if err := h.artLikeService.ArticleCancelLike(c.Request.Context(), userCtx.UserID, req.ArticleID); err != nil {
		c.Error(err)
	}

	common.OK(c, "取消点赞成功", nil)
}

// 用户点赞评论接口
func (h *LikeHandler) CommentLikeHandler(c *gin.Context) {
	// 1. 解析请求的评论 ID
	var req like.CommentIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrCommentNotFound)
		return
	}

	// 2. 从 Gin 上下文获取当前登录用户
	userCtx := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 点赞评论，直接操作redis
	if err := h.comLikeService.CommentLike(c.Request.Context(), userCtx.UserID, req.CommentID); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "点赞评论成功", nil)
}

// 用户取消点赞评论接口
func (h *LikeHandler) CommentCancelLikeHandler(c *gin.Context) {
	// 1. 解析请求的评论 ID
	var req like.CommentIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrCommentNotFound)
		return
	}
	// 2. 从 Gin 上下文获取当前登录用户
	userCtx := c.MustGet("currentUser").(*auth.UserContext)
	// 3. 取消点赞评论，直接操作redis
	if err := h.comLikeService.CommentCancelLike(c.Request.Context(), userCtx.UserID, req.CommentID); err != nil {
		c.Error(err)
		return
	}

	common.OK(c, "取消点赞评论成功", nil)
}
