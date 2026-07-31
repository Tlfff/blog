package cmd

import (
	"blog/config"
	"blog/internal/common"
	"blog/internal/mq"
	"blog/internal/repository"
	"blog/internal/service"
	"blog/pkg/database"
	"blog/pkg/kafka"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
		common.InitValidator()
		// 2. 初始化 MySQL
		db, err := database.NewMySQLClient(
			cfg.Database.Username,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName,
		)
		if err != nil {
			log.Fatalf("初始化 MySQL 失败: %v", err)
		}
		// 安全防御打印
		if db == nil {
			fmt.Println("[error]: NewMySQLClient 返回的 db 对象居然是空的，请检查 pkg/database 里的内部实现！")
			return
		}
		// 3. 初始化 MongoDB
		mongodb, err := database.NewMongoDBClient(
			cfg.Mongodb.Username,
			cfg.Mongodb.Password,
			cfg.Mongodb.Host,
			cfg.Mongodb.DBName,
			cfg.Mongodb.Port,
		)
		if err != nil {
			log.Fatalf("初始化 MongoDB 失败: %v", err)
		}

		// 4. 初始化 Redis
		rdb, err := database.NewRedisClient(cfg.Redis)
		if err != nil {
			log.Fatalf("初始化 Redis 失败: %v", err)
		}
		defer rdb.Close()

		// 5. 初始化 Repository
		userRepo := repository.NewUserRepository(db)
		artRepo := repository.NewArticleRepository(db)
		artLikeRepo := repository.NewArticleLikeRepository(db)
		ntfRepo := repository.NewNotificationRepository(mongodb)
		historyRepo := repository.NewArticleViewHistoryRepository(db)

		// 6. 初始化 Service
		ntfService := service.NewNotificationService(ntfRepo)
		artLikeService := service.NewArticleLikeService(artLikeRepo, artRepo, rdb, ntfService, userRepo, nil)
		historyService := service.NewArticleViewHistoryService(historyRepo, nil)

		// 7. 创建 Kafka 客户端（只用于消费）
		kafkaClient, err := kafka.NewClient(cfg.Kafka)
		if err != nil {
			log.Fatalf("创建 Kafka 客户端失败: %v", err)
		}
		defer kafkaClient.Close()

		// 8. 注册消息处理器
		handlers := mq.RegisterHandlers(artLikeService, historyService)

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
