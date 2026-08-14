package routes

import (
	"blog/internal/handler"
	"blog/internal/middleware"
	"blog/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

func InitArticlePublicRoutes(r *gin.RouterGroup, articleHandler *handler.ArticleHandler) {
	// 获取已发表文章列表
	r.GET("/article/list", articleHandler.GetPublishedList)
	r.GET("/article/hot-rank", articleHandler.GetHotArticleRank)
}
func InitArticleOptionalRoutes(r *gin.RouterGroup, articleHandler *handler.ArticleHandler, historyService *service.ArticleViewHistoryService) {
	// 获取已发表文章详情，浏览历史由中间件异步发送到 Kafka
	r.GET("/article/detail",
		middleware.ViewHistoryMiddleware(historyService),
		articleHandler.GetArticleDetail,
	)
}
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
