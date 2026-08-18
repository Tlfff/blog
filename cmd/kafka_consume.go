package cmd

import (
	communityapp "blog/internal/application/community"
	"blog/internal/infrastructure/bootstrap"
	communityinfra "blog/internal/infrastructure/community"
	"blog/internal/infrastructure/config"
	grpcclient "blog/internal/interfaces/grpc/client"
	mq "blog/internal/interfaces/mq"
	internalv1 "blog/shared/contracts/gen/internalv1"
	svcconfig "blog/shared/platform/config"
	platformgrpc "blog/shared/platform/grpc"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var kafkaConsumeCmd = &cobra.Command{
	Use:   "kafka-consume",
	Short: "启动 Kafka 消费者进程",
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("Kafka 消费者进程开始运行,监听窗口：%s...\n", port)
		// 1. 加载配置
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			log.Fatalf("加载配置失败: %v", err)
			return
		}
		// 1.1 初始化自定义验证器
		bootstrap.InitValidator()
		// 2. 初始化 MySQL
		db, err := bootstrap.NewMySQL(cfg)
		if err != nil {
			log.Fatalf("初始化 MySQL 失败: %v", err)
		}
		// 安全防御打印
		if db == nil {
			fmt.Println("[error]: NewMySQLClient 返回的 db 对象居然是空的，请检查 pkg/database 里的内部实现！")
			return
		}
		// 3. 初始化 MongoDB
		mongodb, err := bootstrap.NewMongoDB(cfg)
		if err != nil {
			log.Fatalf("初始化 MongoDB 失败: %v", err)
		}

		// 4. 初始化 Redis
		rdb, err := bootstrap.NewRedis(cfg)
		if err != nil {
			log.Fatalf("初始化 Redis 失败: %v", err)
		}
		defer rdb.Close()

		// 6. 初始化 Service
		identitySvcCfg, err := svcconfig.Load("services/identity/config.yaml")
		if err != nil {
			log.Fatalf("加载 Identity 服务配置失败: %v", err)
		}
		identityConn, err := platformgrpc.Dial(identitySvcCfg.GRPCAddr, "community-service", 3*time.Second)
		if err != nil {
			log.Fatalf("连接 Identity Service 失败: %v", err)
		}
		defer identityConn.Close()
		communityService := communityapp.NewService(
			nil,
			nil,
			nil,
			communityinfra.NewViewHistoryRepository(db),
			communityinfra.NewNotificationRepository(mongodb),
			communityinfra.NewArticleQuery(db),
			grpcclient.NewCommunityUserInfoClient(internalv1.NewIdentityServiceClient(identityConn)),
			nil,
			nil,
			nil,
			nil,
		)

		// 7. 创建 Kafka 客户端（只用于消费）
		kafkaClient, err := bootstrap.NewKafka(cfg)
		if err != nil {
			log.Fatalf("创建 Kafka 客户端失败: %v", err)
		}
		defer kafkaClient.Close()

		// 8. 注册消息处理器
		handlers := mq.RegisterHandlers(communityService, communityService)

		// 9. 初始化消费者
		if err := kafkaClient.InitConsumer(handlers); err != nil {
			log.Fatalf("初始化消费者失败: %v", err)
		}

		// 10. 如果要关闭 Kafka 客户端
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// 10.1 signal.Notify 捕获操作系统退出信号
		// syscall.SIGINT：按下键盘 Ctrl + C 触发，syscall.SIGTERM：docker/k8s 容器、systemd、运维脚本正常 kill 进程发送的终止信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		// 10.2 开启独立后台监听信号通道，关闭客户端资源
		go func() {
			//用户按下 Ctrl+C / 容器 kill时触发
			<-sigChan
			log.Println("收到退出信号，正在停止消费者...")
			cancel()
			if err := kafkaClient.StopConsumer(); err != nil {
				log.Printf("停止消费者失败: %v", err)
			}
		}()

		// 11. 启动消费者（阻塞）
		log.Println("Kafka 消费者已启动，等待消息...")
		if err := kafkaClient.StartConsumer(ctx); err != nil {
			log.Fatalf("消费者运行异常: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(kafkaConsumeCmd)
	// 2. 绑定端口参数，默认8080
	// 参数含义：1、变量指针：命令行传入的值存在这，2、长选项名，3、短选项名，4、默认值，5、帮助描述
	kafkaConsumeCmd.Flags().StringVarP(&port, "port", "p", "9090", "指定消费者监听端口")
}
