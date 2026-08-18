// Package routes 负责 HTTP 路由注册，按是否需要登录与角色划分路由组。
package routes

import (
	"blog/internal/interfaces/http/handler"
	"blog/internal/interfaces/http/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// 注册文章公开路由（游客可访问，无需登录）
func InitArticlePublicRoutes(r *gin.RouterGroup, articleHandler *handler.ArticleHandler) {
	// 获取已发表文章列表
	r.GET("/article/list", articleHandler.GetPublishedList)
	r.GET("/article/hot-rank", articleHandler.GetHotArticleRank)
}

// 注册文章可选登录路由，登录与未登录都能访问但返回内容有差异
func InitArticleOptionalRoutes(r *gin.RouterGroup, articleHandler *handler.ArticleHandler, historyService middleware.ViewHistorySender) {
	// 获取已发表文章详情，浏览历史由中间件异步发送到 Kafka
	r.GET("/article/detail",
		middleware.ViewHistoryMiddleware(historyService),
		articleHandler.GetArticleDetail,
	)
}

// 注册文章管理路由（需要登录且具备管理员权限）
func InitArticlePrivateRoutes(r *gin.RouterGroup, articleHandler *handler.ArticleHandler) {
	//  创建文章,需要防重复
	r.POST("/article/create",
		middleware.DuplicateMitigation(2*time.Second),
		articleHandler.CreateArticle,
	)
	// 更新文章
	r.POST("/article/update", articleHandler.UpdateArticle)
	// 删除文章（移入垃圾箱）
	r.POST("/article/delete", articleHandler.DeleteArticle)
	// 发表文章
	r.POST("/article/publish", articleHandler.PublishArticle)
	// 获取文章列表
	r.GET("/article/list", articleHandler.GetAdminList)
	// 获取文章详情
	r.GET("/article/me/detail", articleHandler.GetArticleDetailForMe)

	// 以下为垃圾箱相关路由：软删除后的文章管理
	//------------------------------------------------------
	// 获取垃圾箱列表
	r.GET("/article/trash/list", articleHandler.GetTrashList)
	// 恢复文章
	r.POST("/article/trash/recover", articleHandler.RecoverArticle)
	// 硬删除文章
	r.POST("/article/trash/clear", articleHandler.ClearArticle)
	// 获取文章图片上传凭证
	r.POST("/article/image/upload-url", articleHandler.GetImageUploadURL)
}
