package cmd

import (
	"blog/config"
	"blog/internal/auth"
	grpchandler "blog/internal/grpc/handler"
	grpcserver "blog/internal/grpc/server"
	"blog/internal/repository"
	"blog/internal/service"
	"blog/pkg/database"
	"fmt"
	"net"

	"github.com/spf13/cobra"
)

var grpcPort string

var grpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "启动二方gRPC服务",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 加载配置文件
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			fmt.Printf("[error]:加载配置文件失败：%v\n", err)
			return
		}
		// 2. 注入二方服务专用JWT密钥
		auth.InitOpenJWT(cfg.OpenJWT.Secret)

		// 3. 初始化数据库连接
		db, err := database.NewMySQLClient(cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
		if err != nil {
			fmt.Printf("[error]:数据库连接初始化失败：%v\n", err)
			return
		}

		// 4. 初始化Redis连接
		rdb, err := database.NewRedisClient(cfg.Redis)
		if err != nil {
			fmt.Printf("[error]:Redis连接初始化失败：%v\n", err)
			return
		}
		defer rdb.Close()

		// 5. 初始化repository
		userRepo := repository.NewUserRepository(db)
		artRepo := repository.NewArticleRepository(db)
		commentRepo := repository.NewCommentRepository(db)

		// 6. 初始化service
		// gRPC 二方服务用不到 OSS 功能，不注入 OSS
		userService := service.NewUserService(userRepo)
		commentService := service.NewCommentService(commentRepo, artRepo, rdb)
		// 二方服务用不到文章点赞逻辑，artLikeService 传 nil
		artService := service.NewArticleService(artRepo, nil, rdb)

		// 7. 初始化gRPC handler
		userHandler := grpchandler.NewUserHandler(userService)
		commentHandler := grpchandler.NewCommentHandler(commentService)
		articleHandler := grpchandler.NewArticleHandler(artService)

		// 8. 组装gRPC Server（含统一认证拦截器：二方JWT / 三方HMAC）
		s := grpcserver.NewGRPCServer(&grpcserver.AppHandler{
			Article: articleHandler,
			User:    userHandler,
			Comment: commentHandler,
		}, rdb, cfg.ThirdParty)

		// 9. 监听端口（优先命令行参数，默认读config）
		port := grpcPort
		if port == "" {
			port = cfg.GRPC.Port
		}
		lis, err := net.Listen("tcp", ":"+port)
		if err != nil {
			fmt.Printf("[error]:gRPC监听端口失败：%v\n", err)
			return
		}
		fmt.Printf("二方gRPC服务启动，监听端口：%s\n", port)
		if err := s.Serve(lis); err != nil {
			fmt.Printf("[error]:gRPC服务启动失败：%v\n", err)
		}
	},
}

func init() {
	// 1. 将grpc注册到root下
	rootCmd.AddCommand(grpcCmd)
	// 2. 绑定端口参数，默认走config中的grpc.port
	grpcCmd.Flags().StringVarP(&grpcPort, "port", "p", "", "指定gRPC监听端口，默认读取config中的grpc.port")
}
