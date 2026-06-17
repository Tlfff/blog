package routes

import (
	"net/http"

	"blog/internal/handler"
	"blog/internal/middleware"
	"time"
)

// 注册文章相关路由
func InitArticleRoutes(mux *http.ServeMux, articleHandler *handler.ArticleHandler) {

	duplicate2s := middleware.DuplicateMitigation(2 * time.Second)
	// ----------------------- 需要登录的接口 -----------------------

	// 创建文章：先验证登录 -> 再验证防重 -> 最后执行创建
	mux.Handle("POST /article/create", wrap(
		articleHandler.CreateArticle,
		middleware.AuthMiddleware,
		duplicate2s,
	))

	// 修改文章：只需要验证登录
	mux.Handle("POST /article/update", wrap(
		articleHandler.UpdateArticle,
		middleware.AuthMiddleware,
	))

	// 删除文章：只需要验证登录
	mux.Handle("POST /article/delete", wrap(
		articleHandler.DeleteArticle,
		middleware.AuthMiddleware,
	))

	// 发表文章：只需要验证登录
	mux.Handle("POST /article/publish", wrap(
		articleHandler.PublishArticle,
		middleware.AuthMiddleware,
	))

	// 获取草稿列表：只需要验证登录
	mux.Handle("POST /article/draft/list", wrap(
		articleHandler.GetDraftedList,
		middleware.AuthMiddleware,
	))
	// 获取文章详情（私密）
	mux.Handle("GET /article/me/detail", wrap(
		articleHandler.GetArticleDetailForMe,
		middleware.AuthMiddleware,
	))
	// --------------------------- 公开接口 ---------------------------
	// 获取文章详情（公开）
	mux.HandleFunc(
		"GET /article/detail",
		articleHandler.GetArticleDetail,
	)

	// 获取已发表文章列表（公开）
	mux.HandleFunc(
		"GET /article/list",
		articleHandler.GetPublishedList,
	)

}
