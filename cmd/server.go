package cmd

import (
	"blog/internal/handler"
	"blog/internal/repository"
	"blog/internal/routes"
	"blog/internal/service"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var port string
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动博客系统web服务",
	// Long:  `blog server --port 9000`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("博客系统后端服务正在启动,监听窗口：%s...\n", port)
		// 1. 初始化用户模块
		userRepo := repository.NewUserRepository()
		userAuthService := service.NewUserAuthService(userRepo)
		userService := service.NewUserService(userRepo)
		userAuthHandler := handler.NewUserAuthHandler(userAuthService)
		userHandler := handler.NewUserHandler(userService)

		// 2. 初始化文章模块
		articleRepo := repository.NewArticleRepository()
		articleService := service.NewArticleService(articleRepo)
		articleHandler := handler.NewArticleHandler(articleService)

		// 3. 组装成统一的路由容器
		appHandler := &routes.AppHandler{
			UserAuth: userAuthHandler,
			User:     userHandler,
			Article:  articleHandler,
		}

		// 4. 创建路由引擎
		r := gin.New()
		routes.InitRoute(r, appHandler)

		if err := r.Run(":" + port); err != nil {
			fmt.Printf("服务器启动失败：%v\n", err)
		}
	},
}

func init() {
	// 1. 将server注册到root下
	rootCmd.AddCommand(serverCmd)
	// 2. 绑定端口参数，默认8080
	// 参数含义：1、变量指针：命令行传入的值存在这，2、长选项名，3、短选项名，4、默认值，5、帮助描述
	serverCmd.Flags().StringVarP(&port, "port", "p", "8080", "指定服务器监听端口")
}
