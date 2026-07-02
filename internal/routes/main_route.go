package routes

import (
	"blog/internal/handler"
	"blog/internal/middleware"

	"github.com/gin-gonic/gin"
)

// 未来增加新模块，只需在这里加一行，不需要改 InitRoute 的签名
type AppHandler struct {
	UserAuth *handler.UserAuthHandler
	Article  *handler.ArticleHandler
	User     *handler.UserHandler
}

func InitRoute(r *gin.Engine, appHandler *AppHandler) {
	// 1. 挂载全局错误中间件
	r.Use(middleware.GlobalErrorMiddleware())
	// 2.不需要登录的接口
	publicGroup := r.Group("")
	{
		InitArticlePublicRoutes(publicGroup, appHandler.Article)
		InitUserPublicRoutes(publicGroup, appHandler.UserAuth, appHandler.User)
	}
	// 3.需要登录的接口
	privateGroup := r.Group("/my")
	privateGroup.Use(middleware.AuthMiddleware())
	{
		InitUserPrivateRoutes(privateGroup, appHandler.User)
	}
	// 4.管理员管理的接口
	authGroup := r.Group("/admin")
	authGroup.Use(middleware.AuthMiddleware(), middleware.AdminCheckMiddleware())
	{
		InitArticlePrivateRoutes(authGroup, appHandler.Article)
	}

}
