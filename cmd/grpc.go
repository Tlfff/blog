package cmd

import (
	"blog/internal/platform/bootstrap"
	"blog/internal/platform/config"
	grpcserver "blog/internal/platform/interfaces/grpc/server"
	platformsecurity "blog/internal/platform/security"
	"fmt"
	"net"

	"github.com/spf13/cobra"
)

var grpcPort string

var grpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "启动二方gRPC服务",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 加载配置并初始化二方 JWT
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载配置文件失败：%v\n", err)
			return
		}
		platformsecurity.InitOpenJWT(cfg.OpenJWT.Secret)

		// 2. 初始化 gRPC 入口需要的技术资源和业务模块
		resources, err := bootstrap.NewResources(cfg, bootstrap.ResourceOptions{MySQL: true, Redis: true})
		if err != nil {
			fmt.Printf("[error]:平台资源初始化失败：%v\n", err)
			return
		}
		defer closeResources(resources)
		application, err := bootstrap.NewApplication(resources, cfg)
		if err != nil {
			fmt.Printf("[error]:业务模块初始化失败：%v\n", err)
			return
		}

		// 3. 创建 gRPC Adapter 并注册现有 Service
		server := grpcserver.NewGRPCServer(&grpcserver.AppHandler{
			User:    application.User.GRPC,
			Article: application.Article.GRPC,
			Comment: application.Comment.GRPC,
		}, resources.Redis, cfg.ThirdParty)

		// 4. 监听配置或命令行指定端口
		listenPort := grpcPort
		if listenPort == "" {
			listenPort = cfg.GRPC.Port
		}
		listener, err := net.Listen("tcp", ":"+listenPort)
		if err != nil {
			fmt.Printf("[error]:gRPC监听端口失败：%v\n", err)
			return
		}
		fmt.Printf("二方gRPC服务启动，监听端口：%s\n", listenPort)
		if err := server.Serve(listener); err != nil {
			fmt.Printf("[error]:gRPC服务启动失败：%v\n", err)
		}
	},
}

// init 注册 gRPC 子命令及端口参数。
func init() {
	rootCmd.AddCommand(grpcCmd)
	grpcCmd.Flags().StringVarP(&grpcPort, "port", "p", "", "指定gRPC监听端口，默认读取config中的grpc.port")
}
