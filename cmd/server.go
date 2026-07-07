package cmd

import (
	"blog/config"
	"blog/internal/common"
	"blog/internal/handler"
	"blog/internal/repository"
	"blog/internal/routes"
	"blog/internal/service"
	"blog/pkg/database"
	"blog/pkg/iputil"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
		// 1. 加载配置文件
		config, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			fmt.Printf("加载配置文件失败：%v\n", err)
			return
		}
		// 1.1 初始化自定义验证器
		common.InitValidator()
		// 1.2 初始化ip工具类
		dir, _ := os.Getwd() // 获取当前程序运行的绝对路径
		dbPath := filepath.Join(dir, "pkg/resource/ip2region.xdb")
		if err := iputil.InitIPSearcher(dbPath); err != nil {
			log.Fatalf("初始化 IP 解析器失败: %v", err)
		}
		// 在程序退出时，释放内存
		defer iputil.Close()

		// 2. 初始化数据库连接
		db, err := database.NewMySQLClient(config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.DBName)
		if err != nil {
			fmt.Printf("[error]:数据库连接初始化失败：%v\n", err)
			return // 连接失败必须立刻拦截并退出，不能往下传 nil！
		}

		// 3. 安全防御打印
		if db == nil {
			fmt.Println("[error]: NewMySQLClient 返回的 db 对象居然是空的，请检查 pkg/database 里的内部实现！")
			return
		}
		// 4. 初始化模块
		// 4.1  初始化用户模块
		userRepo := repository.NewUserRepository(db)
		userAuthService := service.NewUserAuthService(userRepo)
		userService := service.NewUserService(userRepo)
		userAuthHandler := handler.NewUserAuthHandler(userAuthService)
		userHandler := handler.NewUserHandler(userService)

		// 4.2 初始化文章浏览历史模块
		historyRepo := repository.NewArticleViewHistoryRepository(db)
		historyService := service.NewArticleViewHistoryService(historyRepo)

		// 4.3 初始化文章模块
		articleRepo := repository.NewArticleRepository(db)
		articleService := service.NewArticleService(articleRepo, historyService)
		articleHandler := handler.NewArticleHandler(articleService)
		// 4.4 初始化评论模块
		commentRepo := repository.NewCommentRepository(db)
		commentService := service.NewCommentService(commentRepo, articleRepo)
		commentHandler := handler.NewCommentHandler(commentService)

		// 5. 组装成统一的路由容器
		appHandler := &routes.AppHandler{
			UserAuth: userAuthHandler,
			User:     userHandler,
			Article:  articleHandler,
			Comment:  commentHandler,
		}

		// 6. 创建路由引擎
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
