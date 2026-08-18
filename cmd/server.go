package cmd

import (
	"blog/internal/cron"
	"blog/internal/infrastructure/bootstrap"
	"blog/internal/infrastructure/config"
	grpcclient "blog/internal/interfaces/grpc/client"
	handler "blog/internal/interfaces/http/handler"
	"blog/internal/interfaces/http/routes"
	internalv1 "blog/shared/contracts/gen/internalv1"
	svcconfig "blog/shared/platform/config"
	platformgrpc "blog/shared/platform/grpc"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var port string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动博客系统统一 HTTP 入口",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("统一 HTTP 入口正在启动，监听窗口：%s...\n", port)
		config, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载配置文件失败：%v\n", err)
			return
		}
		bootstrap.InitValidator()

		rdb, err := bootstrap.NewRedis(config)
		if err != nil {
			fmt.Printf("[error]:Redis连接初始化失败:%v\n", err)
			return
		}
		defer rdb.Close()

		identitySvcCfg, err := svcconfig.Load("services/identity/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载 Identity 服务配置失败：%v\n", err)
			return
		}
		if !identitySvcCfg.Enabled {
			fmt.Println("[error]: Identity Service 路由未启用，请先确认服务健康后再开启")
			return
		}
		contentSvcCfg, err := svcconfig.Load("services/content/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载 Content 服务配置失败：%v\n", err)
			return
		}
		if !contentSvcCfg.Enabled {
			fmt.Println("[error]: Content Service 路由未启用，请先确认服务健康后再开启")
			return
		}
		communitySvcCfg, err := svcconfig.Load("services/community/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载 Community 服务配置失败：%v\n", err)
			return
		}
		if !communitySvcCfg.Enabled {
			fmt.Println("[error]: Community Service 路由未启用，请先确认服务健康后再开启")
			return
		}

		identityConn, err := platformgrpc.Dial(identitySvcCfg.GRPCAddr, "gateway-service", 3*time.Second)
		if err != nil {
			fmt.Printf("[error]:连接 Identity Service 失败：%v\n", err)
			return
		}
		defer identityConn.Close()
		contentConn, err := platformgrpc.Dial(contentSvcCfg.GRPCAddr, "gateway-service", 3*time.Second)
		if err != nil {
			fmt.Printf("[error]:连接 Content Service 失败：%v\n", err)
			return
		}
		defer contentConn.Close()
		communityConn, err := platformgrpc.Dial(communitySvcCfg.GRPCAddr, "gateway-service", 3*time.Second)
		if err != nil {
			fmt.Printf("[error]:连接 Community Service 失败：%v\n", err)
			return
		}
		defer communityConn.Close()

		identityService := grpcclient.NewIdentityClient(internalv1.NewIdentityServiceClient(identityConn))
		contentClient := grpcclient.NewContentClient(internalv1.NewContentServiceClient(contentConn))
		communityClient := grpcclient.NewCommunityClient(internalv1.NewCommunityServiceClient(communityConn))

		userAuthHandler := handler.NewUserAuthHandler(identityService)
		userHandler := handler.NewUserHandler(identityService, identityService)
		articleHandler := handler.NewArticleHandler(contentClient, communityClient)
		commentHandler := handler.NewCommentHandler(communityClient)
		likeHandler := handler.NewLikeHandler(communityClient, communityClient)
		ntfHandler := handler.NewNotificationHandler(communityClient)

		rankJob := cron.NewRankSyncJob(communityClient)
		cronMgr := cron.NewCronManager(rankJob)
		cronMgr.Start()
		defer cronMgr.Stop()

		appHandler := &routes.AppHandler{
			UserAuth:    userAuthHandler,
			User:        userHandler,
			Article:     articleHandler,
			Comment:     commentHandler,
			Like:        likeHandler,
			Notify:      ntfHandler,
			ViewHistory: communityClient,
			Redis:       rdb,
		}

		r := gin.New()
		routes.InitRoute(r, appHandler)
		if err := r.Run(":" + port); err != nil {
			fmt.Printf("服务器启动失败：%v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&port, "port", "p", "8080", "指定服务器监听端口")
}
