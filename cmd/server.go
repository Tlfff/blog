package cmd

import (
	"blog/internal/platform/bootstrap"
	"blog/internal/platform/config"
	platformcron "blog/internal/platform/cron"
	"blog/internal/platform/interfaces/http/routes"
	"blog/internal/platform/interfaces/http/validation"
	iputil "blog/internal/platform/ip"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var port string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动博客系统web服务",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("博客系统后端服务正在启动,监听窗口：%s...\n", port)

		// 1. 加载配置并初始化协议级公共能力
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载配置文件失败：%v\n", err)
			return
		}
		validation.InitValidator()
		dir, _ := os.Getwd()
		if err := iputil.InitIPSearcher(filepath.Join(dir, "internal/platform/ip/resource/ip2region.xdb")); err != nil {
			log.Fatalf("[error]:初始化 IP 解析器失败: %v", err)
		}
		defer iputil.Close()

		// 2. 初始化 HTTP 入口需要的技术资源
		resources, err := bootstrap.NewResources(cfg, bootstrap.ResourceOptions{
			MySQL:                 true,
			MongoDB:               true,
			Redis:                 true,
			Kafka:                 true,
			OSS:                   true,
			AllowMongoDBInitError: true,
		})
		if err != nil {
			fmt.Printf("[error]:平台资源初始化失败：%v\n", err)
			return
		}
		defer closeResources(resources)

		// 3. 按上下文依赖顺序组装模块化单体
		application, err := bootstrap.NewApplication(resources, cfg)
		if err != nil {
			fmt.Printf("[error]:业务模块初始化失败：%v\n", err)
			return
		}

		// 4. 保持启动时热榜初始化和 Cron 行为
		initCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := application.Article.Engagement.RebuildHotRank(initCtx); err != nil {
			log.Printf("[WARN] 热门文章排行榜初始化失败或超时, err: %v", err)
		}
		cancel()
		rankJob := platformcron.NewRankSyncJob(application.Article.Engagement)
		cronManager := platformcron.NewCronManager(rankJob)
		cronManager.Start()
		defer cronManager.Stop()

		// 5. 注册各上下文 Module 暴露的 HTTP Adapter
		router := gin.New()
		routes.InitRoute(router, &routes.AppHandler{
			UserAuth:    application.User.HTTPAuth,
			User:        application.User.HTTP,
			Article:     application.Article.HTTP,
			Comment:     application.Comment.HTTP,
			Like:        application.Like.HTTP,
			Notify:      application.Notification.HTTP,
			ViewHistory: application.Article.Engagement,
			Redis:       resources.Redis,
		})

		// 6. 启动 HTTP 服务
		if err := router.Run(":" + port); err != nil {
			fmt.Printf("服务器启动失败：%v\n", err)
		}
	},
}

// closeResources 关闭进程持有的平台资源。
func closeResources(resources *bootstrap.Resources) {
	if err := resources.Close(); err != nil {
		log.Printf("[WARN] 平台资源关闭失败: %v", err)
	}
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&port, "port", "p", "8080", "指定服务器监听端口")
}
