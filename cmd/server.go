package cmd

import (
	communityapp "blog/internal/application/community"
	contentapp "blog/internal/application/content"
	identityapp "blog/internal/application/identity"
	"blog/internal/cron"
	"blog/internal/auth"
	"blog/internal/infrastructure/bootstrap"
	communityinfra "blog/internal/infrastructure/community"
	"blog/internal/infrastructure/config"
	contentinfra "blog/internal/infrastructure/content"
	identityinfra "blog/internal/infrastructure/identity"
	handler "blog/internal/interfaces/http/handler"
	"blog/internal/interfaces/http/routes"
	"blog/internal/repository"
	"context"
	"fmt"
	"log"
	"time"

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
		bootstrap.InitValidator()
		// 1.2 初始化ip工具类
		if err := bootstrap.InitIPSearcher(); err != nil {
			log.Fatalf("[error]:初始化 IP 解析器失败: %v", err)
		}
		// 在程序退出时，释放内存
		defer bootstrap.CloseIP()

		// 2 初始化数据库连接
		// 2.1 初始化mysql
		db, err := bootstrap.NewMySQL(config)
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
		mongodb, err := bootstrap.NewMongoDB(config)
		if err != nil {
			fmt.Printf("[error]:MongoDB连接初始化失败:%v\n", err)
			return
		}

		// 3. 初始化redis连接
		rdb, err := bootstrap.NewRedis(config)
		if err != nil {
			fmt.Printf("[error]:Redis连接初始化失败:%v\n", err)
			return
		}
		defer rdb.Close()
		// 3.1 初始化Kafka客户端
		kafkaClient, err := bootstrap.NewKafka(config)
		if err != nil {
			fmt.Printf("[error]:Kafka客户端初始化失败:%v\n", err)
			return
		}
		// 如果初始化成功，在程序退出时关闭
		if kafkaClient != nil {
			defer kafkaClient.Close()
		}

		// 4. 初始化模块
		// 4.1  初始化基础 Repository
		userRepo := repository.NewUserRepository(db)
		historyRepo := repository.NewArticleViewHistoryRepository(db)
		artRepo := repository.NewArticleRepository(db)
		commentRepo := repository.NewCommentRepository(db)
		artLikeRepo := repository.NewArticleLikeRepository(db)
		commentLikeRepo := repository.NewCommentLikeRepository(db)
		ntfRepo := repository.NewNotificationRepository(mongodb)

		// 4.1.1 初始化 MinIO 客户端
		ossClient, err := bootstrap.NewOSS(config)
		if err != nil {
			fmt.Printf("[error]:MinIO客户端初始化失败:%v\n", err)
			return
		}

		// 4.2 初始化service
		tokenAuth := auth.NewTokenAuth(rdb)
		identityService := identityapp.NewService(
			identityinfra.NewUserRepository(userRepo),
			identityinfra.NewTokenSession(tokenAuth),
			identityinfra.NewPasswordChangeTokenStore(rdb),
			identityinfra.NewAvatarStorage(ossClient),
			config.OSS.PublicDomain,
			config.OSS.AllowedExts,
		)
		articleLikePort := communityinfra.NewArticleLikeRepository(artLikeRepo)
		commentLikePort := communityinfra.NewCommentLikeRepository(commentLikeRepo)
		communityService := communityapp.NewService(
			communityinfra.NewCommentRepository(commentRepo, artRepo),
			articleLikePort,
			commentLikePort,
			communityinfra.NewViewHistoryRepository(historyRepo),
			communityinfra.NewNotificationRepository(ntfRepo),
			communityinfra.NewArticleQuery(artRepo),
			communityinfra.NewUserInfoQuery(userRepo),
			communityinfra.NewLikeCache(rdb, articleLikePort, commentLikePort),
			communityinfra.NewLikeCountStore(rdb),
			communityinfra.NewHotRankStore(rdb),
			communityinfra.NewEventPublisher(kafkaClient),
		)
		contentService := contentapp.NewService(
			contentinfra.NewArticleRepository(artRepo),
			contentinfra.NewArticleImageStorage(ossClient),
			communityService,
			config.OSS.PublicDomain,
			config.OSS.AllowedExts,
		)

		// 初始化排行榜
		initCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := communityService.RebuildHotRank(initCtx); err != nil {
			log.Printf("[WARN] 热门文章排行榜初始化失败或超时, err: %v", err)
		}
		cancel() //释放定时器资源

		// 4.3 初始化handler
		userAuthHandler := handler.NewUserAuthHandler(identityService)
		userHandler := handler.NewUserHandler(identityService, identityService)
		articleHandler := handler.NewArticleHandler(contentService, communityService)
		commentHandler := handler.NewCommentHandler(communityService)
		likeHandler := handler.NewLikeHandler(communityService, communityService)
		ntfHandler := handler.NewNotificationHandler(communityService)

		// 4.4 初始化定时器
		// likeSyncJob := cron.NewLikeSyncJob(likeService)
		rankJob := cron.NewRankSyncJob(communityService)
		// 传入所有定时任务，由全局管理器统一调度
		cronMgr := cron.NewCronManager(rankJob)
		cronMgr.Start()
		defer cronMgr.Stop()

		// 5. 组装成统一的路由容器
		appHandler := &routes.AppHandler{
			UserAuth:    userAuthHandler,
			User:        userHandler,
			Article:     articleHandler,
			Comment:     commentHandler,
			Like:        likeHandler,
			Notify:      ntfHandler,
			ViewHistory: communityService,
			Redis:       rdb,
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
