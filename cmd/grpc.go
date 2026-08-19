package cmd

import (
	"blog/internal/platform/bootstrap"
	"blog/internal/platform/config"
	grpcclient "blog/internal/platform/interfaces/grpc/client"
	grpchandler "blog/internal/platform/interfaces/grpc/handler"
	grpcserver "blog/internal/platform/interfaces/grpc/server"
	internalv1 "blog/shared/contracts/gen/internalv1"
	svcconfig "blog/shared/platform/config"
	platformgrpc "blog/shared/platform/grpc"
	"fmt"
	"net"
	"time"

	"github.com/spf13/cobra"
)

var grpcPort string

var grpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "启动二方gRPC服务",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载配置文件失败：%v\n", err)
			return
		}
		bootstrap.InitOpenJWT(cfg.OpenJWT.Secret)

		rdb, err := bootstrap.NewRedis(cfg)
		if err != nil {
			fmt.Printf("[error]:Redis连接初始化失败：%v\n", err)
			return
		}
		defer rdb.Close()

		identitySvcCfg, err := svcconfig.Load("services/identity/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载 Identity 服务配置失败：%v\n", err)
			return
		}
		contentSvcCfg, err := svcconfig.Load("services/content/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载 Content 服务配置失败：%v\n", err)
			return
		}
		communitySvcCfg, err := svcconfig.Load("services/community/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载 Community 服务配置失败：%v\n", err)
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

		userHandler := grpchandler.NewUserHandler(grpcclient.NewIdentityClient(internalv1.NewIdentityServiceClient(identityConn)))
		articleHandler := grpchandler.NewArticleHandler(grpcclient.NewContentClient(internalv1.NewContentServiceClient(contentConn)))
		commentHandler := grpchandler.NewCommentHandler(grpcclient.NewCommunityClient(internalv1.NewCommunityServiceClient(communityConn)))

		s := grpcserver.NewGRPCServer(&grpcserver.AppHandler{
			Article: articleHandler,
			User:    userHandler,
			Comment: commentHandler,
		}, rdb, cfg.ThirdParty)

		listenPort := grpcPort
		if listenPort == "" {
			listenPort = cfg.GRPC.Port
		}
		lis, err := net.Listen("tcp", ":"+listenPort)
		if err != nil {
			fmt.Printf("[error]:gRPC监听端口失败：%v\n", err)
			return
		}
		fmt.Printf("二方gRPC服务启动，监听端口：%s\n", listenPort)
		if err := s.Serve(lis); err != nil {
			fmt.Printf("[error]:gRPC服务启动失败：%v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(grpcCmd)
	grpcCmd.Flags().StringVarP(&grpcPort, "port", "p", "", "指定gRPC监听端口，默认读取config中的grpc.port")
}
