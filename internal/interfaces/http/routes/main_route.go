package routes

import (
	"blog/internal/interfaces/http/handler"
	"blog/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// 未来增加新模块，只需在这里加一行，不需要改 InitRoute 的签名
type AppHandler struct {
	UserAuth *handler.UserAuthHandler
	Article  *handler.ArticleHandler
	User     *handler.UserHandler
	Comment  *handler.CommentHandler
	Like     *handler.LikeHandler
	Notify   *handler.NotificationHandler
	// 浏览历史服务，路由级中间件使用（不经过 handler）
	ViewHistory middleware.ViewHistorySender
	Redis       *redis.Client
}

func InitRoute(r *gin.Engine, appHandler *AppHandler) {
	// 1. 全局中间件
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.GlobalErrorMiddleware())
	// 2.不需要登录的接口（/r/xxx）
	publicGroup := r.Group("")
	{
		InitArticlePublicRoutes(publicGroup, appHandler.Article)
		InitUserPublicRoutes(publicGroup, appHandler.UserAuth, appHandler.User)
		InitCommentPublicRoutes(publicGroup, appHandler.Comment)
	}

	// 3.需要登录的接口
	privateGroup := r.Group("/auth")
	privateGroup.Use(middleware.MustAuth(appHandler.Redis))
	{
		InitUserPrivateRoutes(privateGroup, appHandler.User)
		InitCommentPrivateRoutes(privateGroup, appHandler.Comment)

		// 点赞功能相关接口
		InitLikePrivateRoutes(privateGroup, appHandler.Like)
		// 通知功能相关接口
		InitNotificationPrivateRoutes(privateGroup, appHandler.Notify)

	}
	// 4.管理员管理的接口
	authGroup := r.Group("/admin")
	authGroup.Use(middleware.MustAuth(appHandler.Redis), middleware.AdminCheckMiddleware())
	{
		InitArticlePrivateRoutes(authGroup, appHandler.Article)
		InitCommentAdminRoutes(authGroup, appHandler.Comment)
	}

	// 5.登录和不登录功能有区别的接口
	optionalGroup := r.Group("/optional")
	optionalGroup.Use(middleware.OptionalAuth(appHandler.Redis))
	{
		InitArticleOptionalRoutes(optionalGroup, appHandler.Article, appHandler.ViewHistory)
	}

}
