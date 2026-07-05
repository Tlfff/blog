package routes

import (
	"blog/internal/handler"
	"blog/internal/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// 公开评论路由 (游客可用)
func InitCommentPublicRoutes(r *gin.RouterGroup, commentHandler *handler.CommentHandler) {
	// 获取文章主评论列表
	r.GET("/comment/list/roots", commentHandler.ListRoots)
	// 展开获取子评论列表 (楼中楼)
	r.GET("/comment/list/replies", commentHandler.ListReplies)
}

// 登录用户路由
func InitCommentPrivateRoutes(r *gin.RouterGroup, commentHandler *handler.CommentHandler) {
	// 创建评论
	r.POST("/comment/create",
		middleware.DuplicateMitigation(2*time.Second),
		commentHandler.Create,
	)

	// 用户删除自己的评论
	r.POST("/comment/delete", commentHandler.DeleteMyComment)
}

// 管理员专属路由
func InitCommentAdminRoutes(r *gin.RouterGroup, commentHandler *handler.CommentHandler) {
	// 管理员强制后台处理/删除违规评论
	r.POST("/comment/delete", commentHandler.DeleteAdminComment)
}
