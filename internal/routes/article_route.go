package routes

import (
	"blog/internal/handler"
	"blog/internal/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

// // 注册文章相关路由
// func InitArticleRoutes(r *gin.Engine, articleHandler *handler.ArticleHandler) {

// 	// ----------------------- 需要登录的接口 -----------------------

// 	// 创建文章：先验证登录 -> 再验证防重 -> 最后执行创建
// 	r.POST("/article/create",
// 		middleware.AuthMiddleware(),
// 		middleware.DuplicateMitigation(2*time.Second),
// 		articleHandler.CreateArticle,
// 	)

// 	// 修改文章：只需要验证登录
// 	r.POST("/article/update",

// 		middleware.AuthMiddleware(),
// 		articleHandler.UpdateArticle,
// 	)

// 	// 删除文章：只需要验证登录
// 	r.POST("/article/delete",

// 		middleware.AuthMiddleware(),
// 		articleHandler.DeleteArticle,
// 	)

// 	// 发表文章：只需要验证登录
// 	r.POST("/article/publish",
// 		middleware.AuthMiddleware(),
// 		articleHandler.PublishArticle,
// 	)

// 	// 获取草稿列表：只需要验证登录
// 	r.POST("/article/draft/list",
// 		middleware.AuthMiddleware(),
// 		articleHandler.GetDraftedList,
// 	)
// 	// 获取文章详情（私密）
// 	r.POST("/article/me/detail",
// 		middleware.AuthMiddleware(),
// 		articleHandler.GetArticleDetailForMe,
// 	)
// 	// --------------------------- 公开接口 ---------------------------
// 	// 获取文章详情（公开）
// 	r.GET(
// 		"/article/detail",
// 		articleHandler.GetArticleDetail,
// 	)

// 	// 获取已发表文章列表（公开）
// 	r.GET(
// 		"/article/list",
// 		articleHandler.GetPublishedList,
// 	)

// }

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
	r.POST("/article/list", articleHandler.GetAdminList)
	// 获取文章详情
	r.POST("/article/me/detail", articleHandler.GetArticleDetailForMe)

	//------------------------------------------------------
	// 获取垃圾箱列表
	r.POST("/article/trash/list", articleHandler.GetTrashList)
	// 恢复文章
	r.POST("/article/trash/recover", articleHandler.RecoverArticle)
	// 硬删除文章
	r.POST("/article/trash/clear", articleHandler.ClearArticle)

}
