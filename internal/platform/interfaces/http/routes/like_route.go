package routes

import (
	"blog/internal/platform/interfaces/http/handler"

	"github.com/gin-gonic/gin"
)

// 点赞路由
func InitLikePrivateRoutes(r *gin.RouterGroup, LikeHandler *handler.LikeHandler) {
	// 点赞文章
	r.POST("/article/like", LikeHandler.ArticleLikeHandler)
	// 取消点赞文章
	r.POST("/article/cancel_like", LikeHandler.ArticleCancelLikeHandler)

	// 点赞评论
	r.POST("/comment/like", LikeHandler.CommentLikeHandler)
	// 取消点赞评论
	r.POST("/comment/cancel_like", LikeHandler.CommentCancelLikeHandler)
}
