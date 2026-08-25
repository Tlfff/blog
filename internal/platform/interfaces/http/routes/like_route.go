package routes

import (
	likehttp "blog/internal/like/interfaces/http"

	"github.com/gin-gonic/gin"
)

// InitLikePrivateRoutes 注册点赞私有路由。
func InitLikePrivateRoutes(r *gin.RouterGroup, LikeHandler *likehttp.LikeHandler) {
	// 点赞文章
	r.POST("/article/like", LikeHandler.ArticleLikeHandler)
	// 取消点赞文章
	r.POST("/article/cancel_like", LikeHandler.ArticleCancelLikeHandler)

	// 点赞评论
	r.POST("/comment/like", LikeHandler.CommentLikeHandler)
	// 取消点赞评论
	r.POST("/comment/cancel_like", LikeHandler.CommentCancelLikeHandler)
}
