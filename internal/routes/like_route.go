package routes

import (
	"blog/internal/handler"

	"github.com/gin-gonic/gin"
)

// 点赞路由
func InitLikePrivateRoutes(r *gin.RouterGroup, LikeHandler *handler.LikeHandler) {
	r.POST("/article/like", LikeHandler.ArticleLikeHandler)
	r.POST("/article/cancel_like", LikeHandler.ArticleCancelLikeHandler)

	r.POST("/comment/like", LikeHandler.CommentLikeHandler)
	r.POST("/comment/cancel_like", LikeHandler.CommentCancelLikeHandler)
}
