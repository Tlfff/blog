// Package routes 负责 HTTP 路由注册，按是否需要登录与角色划分路由组。
package routes

import (
	articlehttp "blog/internal/article/interfaces/http"
	"blog/internal/platform/interfaces/http/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// InitArticlePublicRoutes 注册文章公开路由。
func InitArticlePublicRoutes(r *gin.RouterGroup, articleHandler *articlehttp.ArticleHandler) {
	// 获取已发表文章列表
	r.GET("/article/list", articleHandler.GetPublishedList)
	r.GET("/article/hot-rank", articleHandler.GetHotArticleRank)
}

// InitArticleOptionalRoutes 注册文章可选登录路由。
func InitArticleOptionalRoutes(r *gin.RouterGroup, articleHandler *articlehttp.ArticleHandler, historyService middleware.ViewHistorySender) {
	// 获取已发表文章详情，浏览历史由中间件异步发送到 Kafka
	r.GET("/article/detail",
		middleware.ViewHistoryMiddleware(historyService),
		articleHandler.GetArticleDetail,
	)
}

// InitArticlePrivateRoutes 注册文章管理路由。
func InitArticlePrivateRoutes(r *gin.RouterGroup, articleHandler *articlehttp.ArticleHandler) {
	// 初始化空内容文章草稿，用于提前取得文章 ID
	r.POST("/article/init",
		middleware.DuplicateMitigation(2*time.Second),
		articleHandler.InitializeArticle,
	)
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
	// 批量获取直接写入文章正式目录的图片上传凭证
	r.POST("/article/image/upload-urls", articleHandler.GetImageUploadURLs)
}
