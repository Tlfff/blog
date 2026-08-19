package kafka

import (
	"blog/internal/platform/config"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// DeadLetterHandler 死信消息处理函数类型
// 由业务方实现，用于处理从死信队列消费到的消息
// 参数：
//   - ctx: 上下文
//   - dlMsg: 死信消息（已解析）
//
// 返回 error 表示处理失败；如果返回 error，该消息的 offset 不会提交，会重新消费
type DeadLetterHandler func(ctx context.Context, dlMsg *DeadLetterMessage) error

// DeadLetterConsumer 死信队列消费者
// 独立的消费者组，消费 dead_letter topic 的消息
type DeadLetterConsumer struct {
	reader  *kafka.Reader      // 死信队列的 Reader
	config  config.Kafka       // Kafka 配置
	handler DeadLetterHandler  // 死信消息处理器
	mu      sync.RWMutex       // 读写锁，保护 running 状态
	cancel  context.CancelFunc // 取消函数
	wg      sync.WaitGroup     // 等待消费 goroutine 退出
	running bool               // 是否正在运行
}

// NewDeadLetterConsumer 创建死信队列消费者
//
// 参数：
//   - cfg: Kafka 配置
//   - handler: 死信消息处理器，由业务方实现
func NewDeadLetterConsumer(cfg config.Kafka, handler DeadLetterHandler) (*DeadLetterConsumer, error) {
	// 1. 获取死信队列相关配置：broker 列表、topic 与消费者组

	// 1. 获取配置
	brokers := cfg.GetBrokerList()

	dlqTopic := cfg.GetDeadLetterTopic()

	consumerGroup := cfg.GetDeadLetterConsumerGroup()

	// 2. 创建 Reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          dlqTopic,
		GroupID:        consumerGroup,
		GroupBalancers: parseGroupBalancers(cfg.Consumer.GroupBalancer),
		StartOffset:    kafka.FirstOffset, // 死信队列从头开始消费，不丢消息
		MinBytes:       cfg.Consumer.MinBytes,
		MaxBytes:       cfg.Consumer.MaxBytes,
		MaxWait:        time.Duration(cfg.Consumer.CommitWait) * time.Millisecond,
		SessionTimeout: time.Duration(cfg.Consumer.SessionTimeout) * time.Millisecond,
	})

	// 5. 返回消费者实例
	return &DeadLetterConsumer{
		reader:  reader,
		config:  cfg,
		handler: handler,
		running: false,
	}, nil
}

// 启动死信队列消费者，阻塞直到上下文取消或调用 Stop
// 启动死信队列消费者（阻塞）
func (c *DeadLetterConsumer) Start(ctx context.Context) error {
	// 1. 检查是否已启用
	if c == nil {
		log.Printf("死信队列未启用，跳过启动")
		return nil
	}

	// 2. 检查是否已在运行
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	// 3. 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// 4. 启动消费 goroutine
	c.wg.Add(1)
	go c.consumeLoop(ctx)

	// 5. 等待消费 goroutine 退出
	c.wg.Wait()
	return nil
}

// 循环拉取死信消息，解析元信息后交给业务处理器并提交 offset
// 消费循环
func (c *DeadLetterConsumer) consumeLoop(ctx context.Context) {
	defer c.wg.Done()
	defer c.reader.Close()

	// 1. 循环拉取消息，直到上下文被取消
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("拉取死信消息失败: %v", err)
				continue
			}

			// 1. 从 Headers 中提取元信息
			dlMsg := &DeadLetterMessage{
				Value: string(msg.Value),
				Key:   string(msg.Key),
			}

			for _, header := range msg.Headers {
				switch header.Key {
				case "x-dlq-source-topic":
					dlMsg.SourceTopic = string(header.Value)
				case "x-dlq-source-partition":
					fmt.Sscanf(string(header.Value), "%d", &dlMsg.Partition)
				case "x-dlq-source-offset":
					fmt.Sscanf(string(header.Value), "%d", &dlMsg.Offset)
				case "x-dlq-error":
					dlMsg.Error = string(header.Value)
				case "x-dlq-retries":
					fmt.Sscanf(string(header.Value), "%d", &dlMsg.Retries)
				case "x-dlq-timestamp":
					dlMsg.Timestamp, _ = time.Parse(time.RFC3339, string(header.Value))
				case "x-dlq-consumer-group":
					dlMsg.ConsumerGroup = string(header.Value)
				}
			}

			// 2. 调用业务处理器
			if err := c.handler(ctx, dlMsg); err != nil {
				log.Printf("处理死信消息失败, source_topic: %s, offset: %d, err: %v",
					dlMsg.SourceTopic, dlMsg.Offset, err)
				continue
			}

			// 3. 提交 offset
			c.commitWithRetry(ctx, msg)
		}
	}
}

// 提交 offset 并在失败时按次数递增间隔重试
func (c *DeadLetterConsumer) commitWithRetry(ctx context.Context, msg kafka.Message) {
	commitMaxRetries := c.config.GetDeadLetterMaxRetries()
	commitWait := time.Duration(c.config.Consumer.CommitWait) * time.Millisecond

	for attempt := 1; attempt <= commitMaxRetries; attempt++ {
		if err := c.reader.CommitMessages(ctx, msg); err == nil {
			log.Printf("死信消息提交成功, partition: %d, offset: %d",
				msg.Partition, msg.Offset)
			return
		} else {
			log.Printf("提交死信消息 offset 失败, attempt: %d/%d, err: %v",
				attempt, commitMaxRetries, err)
			if attempt == commitMaxRetries {
				log.Printf("提交死信消息 offset 最终失败, 消息将被重新消费")
				return
			}
			waitForRetry(ctx, time.Duration(attempt)*commitWait)
		}
	}
}

// 停止死信队列消费者，取消上下文并等待消费协程退出
// Stop 停止死信队列消费者
func (c *DeadLetterConsumer) Stop() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	c.wg.Wait()
	c.running = false

	log.Printf("死信队列消费者已停止")
	return nil
}

// 关闭死信队列消费者，等价于 Stop
// Close 关闭死信队列消费者
func (c *DeadLetterConsumer) Close() error {
	return c.Stop()
}

// 检查死信队列消费者是否正在运行
// IsRunning 检查是否正在运行
func (c *DeadLetterConsumer) IsRunning() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}
