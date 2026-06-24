package main

import (
	"blog/internal/handler"
	"blog/internal/repository"
	"blog/internal/routes"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化用户模块
	userRepo := repository.NewUserRepository()
	userService := service.NewUserAuthService(userRepo)
	userHandler := handler.NewUserAuthHandler(userService)

	// 2. 初始化文章模块
	articleRepo := repository.NewArticleRepository()
	articleService := service.NewArticleService(articleRepo)
	articleHandler := handler.NewArticleHandler(articleService)

	// 3. 组装成统一的路由容器
	appHandler := &routes.AppHandler{
		UserAuth: userHandler,
		Article:  articleHandler,
	}

	// 4. 创建路由引擎
	// mux := routes.InitRoute(appHandler)
	// http.ListenAndServe(":8080", mux)
	r := gin.New()
	routes.InitRoute(r, appHandler)
	r.Run(":8080")
}
