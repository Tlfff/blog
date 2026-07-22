package cmd

import (
	"blog/config"
	"blog/internal/common"
	"blog/internal/cron"
	"blog/internal/handler"
	"blog/internal/repository"
	"blog/internal/routes"
	"blog/internal/service"
	"blog/pkg/database"
	iputil "blog/pkg/util/ip"
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
			fmt.Printf("[error]:加载配置文件失败：%v\n", err)
			return
		}
		// 1.1 初始化自定义验证器
		common.InitValidator()
		// 1.2 初始化ip工具类
		dir, _ := os.Getwd() // 获取当前程序运行的绝对路径
		dbPath := filepath.Join(dir, "pkg/resource/ip2region.xdb")
		if err := iputil.InitIPSearcher(dbPath); err != nil {
			log.Fatalf("[error]:初始化 IP 解析器失败: %v", err)
		}
		// 在程序退出时，释放内存
		defer iputil.Close()

		// 2 初始化数据库连接
		// 2.1 初始化mysql
		db, err := database.NewMySQLClient(config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.DBName)
		if err != nil {
			fmt.Printf("[error]:数据库连接初始化失败：%v\n", err)
			return // 连接失败必须立刻拦截并退出，不能往下传 nil！
		}

		// 安全防御打印
		if db == nil {
			fmt.Println("[error]: NewMySQLClient 返回的 db 对象居然是空的，请检查 pkg/database 里的内部实现！")
			return
		}
		// 2.2 初始化mongodb
		mongodb, err := database.NewMongoDBClient(config.Mongodb.Username, config.Mongodb.Password, config.Mongodb.Host, config.Mongodb.DBName, config.Mongodb.Port)

		// 3. 初始化redis连接
		rdb, err := database.NewRedisClient(config.Redis)
		if err != nil {
			fmt.Printf("[error]:Redis连接初始化失败:%v\n", err)
			return
		}
		defer rdb.Close()

		// 4. 初始化模块
		// 4.1  初始化基础 Repository
		userRepo := repository.NewUserRepository(db)
		historyRepo := repository.NewArticleViewHistoryRepository(db)
		articleRepo := repository.NewArticleRepository(db)
		commentRepo := repository.NewCommentRepository(db)
		articleLikeRepo := repository.NewArticleLikeRepository(db)
		commentLikeRepo := repository.NewCommentLikeRepository(db)
		ntfRepo := repository.NewNotificationRepository(mongodb)

		// 4.2 初始化service
		ntfService := service.NewNotificationService(ntfRepo)
		artLikeService := service.NewArticleLikeService(articleLikeRepo, articleRepo, rdb, ntfService, userRepo)
		comLikeService := service.NewCommentLikeService(commentLikeRepo, commentRepo, rdb)
		userAuthService := service.NewUserAuthService(userRepo)
		userService := service.NewUserService(userRepo)
		historyService := service.NewArticleViewHistoryService(historyRepo)
		articleService := service.NewArticleService(articleRepo, historyService, artLikeService, rdb)
		articleRankService := service.NewArticleRankService(articleRepo, rdb)
		commentService := service.NewCommentService(commentRepo, articleRepo, rdb)

		// 4.3 初始化handler
		userAuthHandler := handler.NewUserAuthHandler(userAuthService)
		userHandler := handler.NewUserHandler(userService)
		articleHandler := handler.NewArticleHandler(articleService, articleRankService)
		commentHandler := handler.NewCommentHandler(commentService)
		likeHandler := handler.NewLikeHandler(artLikeService, comLikeService)
		ntfHandler := handler.NewNotificationHandler(ntfService)

		// 4.4 初始化定时器
		// likeSyncJob := cron.NewLikeSyncJob(likeService)
		rankJob := cron.NewRankSyncJob(articleRankService)
		// 传入所有定时任务，由全局管理器统一调度
		cronMgr := cron.NewCronManager(rankJob)
		cronMgr.Start()
		defer cronMgr.Stop()

		// 5. 组装成统一的路由容器
		appHandler := &routes.AppHandler{
			UserAuth: userAuthHandler,
			User:     userHandler,
			Article:  articleHandler,
			Comment:  commentHandler,
			Like:     likeHandler,
			Notify:   ntfHandler,
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
