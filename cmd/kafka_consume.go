package cmd

import (
	"blog/internal/platform/bootstrap"
	"blog/internal/platform/config"
	"blog/internal/platform/interfaces/http/validation"
	platformmq "blog/internal/platform/interfaces/mq"
	"context"
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

		// 1. 加载配置并初始化校验器
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			log.Fatalf("加载配置失败: %v", err)
		}
		validation.InitValidator()

		// 2. 初始化消费者入口需要的技术资源和业务模块
		resources, err := bootstrap.NewResources(cfg, bootstrap.ResourceOptions{
			MySQL: true, MongoDB: true, Redis: true, Kafka: true,
		})
		if err != nil {
			log.Fatalf("初始化平台资源失败: %v", err)
		}
		defer closeResources(resources)
		application, err := bootstrap.NewApplication(resources, cfg)
		if err != nil {
			log.Fatalf("初始化业务模块失败: %v", err)
		}

		// 3. 注册当前基线已接线的通知和浏览历史 Handler
		handlers := platformmq.RegisterHandlers(
			application.Notification.Kafka,
			application.Article.Kafka,
		)
		if err := resources.Kafka.InitConsumer(handlers); err != nil {
			log.Fatalf("初始化消费者失败: %v", err)
		}

		// 4. 监听退出信号并停止 Consumer
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(signalChannel)
		go func() {
			<-signalChannel
			log.Println("收到退出信号，正在停止消费者...")
			cancel()
			if err := resources.Kafka.StopConsumer(); err != nil {
				log.Printf("停止消费者失败: %v", err)
			}
		}()

		// 5. 阻塞运行消费者
		log.Println("Kafka 消费者已启动，等待消息...")
		if err := resources.Kafka.StartConsumer(ctx); err != nil {
			log.Fatalf("消费者运行异常: %v", err)
		}
	},
}

// init 注册 Kafka Consumer 子命令及端口参数。
func init() {
	rootCmd.AddCommand(kafkaConsumeCmd)
	kafkaConsumeCmd.Flags().StringVarP(&port, "port", "p", "9090", "指定消费者监听端口")
}
