package routes

import (
	articlehttp "blog/internal/article/interfaces/http"
	commenthttp "blog/internal/comment/interfaces/http"
	likehttp "blog/internal/like/interfaces/http"
	notificationhttp "blog/internal/notification/interfaces/http"
	"blog/internal/platform/interfaces/http/middleware"
	userhttp "blog/internal/user/interfaces/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AppHandler 汇总所有 HTTP Handler 与路由级依赖，新增模块只需在此追加字段。
type AppHandler struct {
	UserAuth *userhttp.UserAuthHandler             // 用户注册登录 Handler
	Article  *articlehttp.ArticleHandler           // 文章 Handler
	User     *userhttp.UserHandler                 // 用户资料 Handler
	Comment  *commenthttp.CommentHandler           // 评论 Handler
	Like     *likehttp.LikeHandler                 // 点赞 Handler
	Notify   *notificationhttp.NotificationHandler // 通知 Handler
	// 浏览历史服务，路由级中间件使用（不经过 handler）
	ViewHistory middleware.ViewHistorySender // 浏览历史发送器，供浏览历史中间件异步投递事件
	Redis       *redis.Client                // Redis 客户端，供鉴权中间件校验会话
}

// 注册全局中间件并按权限维度挂载全部路由组
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
