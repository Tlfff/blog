package routes

import (
	"blog/internal/handler"
	"blog/internal/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func InitArticlePublicRoutes(r *gin.RouterGroup, articleHandler *handler.ArticleHandler) {
	// 获取已发表文章详情
	r.GET("/article/detail", articleHandler.GetArticleDetail)

	// 获取已发表文章列表
	r.GET("/article/list", articleHandler.GetPublishedList)
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
}
