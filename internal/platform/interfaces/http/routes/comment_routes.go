package routes

import (
	commenthttp "blog/internal/comment/interfaces/http"
	"blog/internal/platform/interfaces/http/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// InitCommentPublicRoutes 注册评论公开路由。
func InitCommentPublicRoutes(r *gin.RouterGroup, commentHandler *commenthttp.CommentHandler) {
	// 获取文章主评论列表
	r.GET("/comment/list/roots", commentHandler.ListRoots)
	// 展开获取子评论列表 (楼中楼)
	r.GET("/comment/list/replies", commentHandler.ListReplies)
}

// InitCommentPrivateRoutes 注册评论私有路由。
func InitCommentPrivateRoutes(r *gin.RouterGroup, commentHandler *commenthttp.CommentHandler) {
	// 创建评论
	r.POST("/comment/create",
		middleware.DuplicateMitigation(2*time.Second),
		commentHandler.Create,
	)

	// 用户删除自己的评论
	r.POST("/comment/delete", commentHandler.DeleteMyComment)
}

// InitCommentAdminRoutes 注册评论管理员路由。
func InitCommentAdminRoutes(r *gin.RouterGroup, commentHandler *commenthttp.CommentHandler) {
	// 管理员强制后台处理/删除违规评论
	r.POST("/comment/delete", commentHandler.DeleteAdminComment)
}
