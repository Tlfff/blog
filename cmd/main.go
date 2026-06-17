package main

import (
	"net/http"

	"blog/internal/handler"
	"blog/internal/repository"
	"blog/internal/routes"
	"blog/internal/service"
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

	// 4. 传给总路由
	mux := routes.InitRoute(appHandler)

	http.ListenAndServe(":8080", mux)
}
