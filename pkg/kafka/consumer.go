package kafka

import (
	"blog/internal/platform/config"
	"blog/internal/shared/common"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

//	消息处理函数类型
//
// 参数：
//   - ctx: 上下文
//   - key: 消息的 key（字符串格式）
//   - value: 消息的 value（原始 JSON 字节数组）
//
// 返回 error 表示本次处理失败；消费者会自动重试，超过最大重试次数后记录并丢弃消息，继续提交后续安全 offset。
type MessageHandler func(ctx context.Context, key string, value []byte) error

const (
	// maxMessageRetries 表示在首次处理失败后，最多再重试 3 次。
	// 因此一条消息最多会被 Handler 调用 4 次：首次处理 + 3 次重试。
	maxMessageRetries = 3
	maxCommitAttempts = 3
)

// 消费者
type Consumer struct {
	readers  map[string]*kafka.Reader  // 保存消费者组
	config   config.Kafka              // 配置
	handlers map[string]MessageHandler // 每个 topic 需要对应的处理器
	mu       sync.RWMutex              // 读写锁
	cancel   context.CancelFunc        // 全局上下文取消函数
	wg       sync.WaitGroup            // 等待所有 Topic 的消费 goroutine 全部正常退出
	running  bool                      // 消费者是否正在运行中
}

//	创建消费者
//
// 参数：
//   - cfg: Kafka 配置
//   - handlers: topicKey -> MessageHandler 的映射，每个 topic 需要对应的处理器
func NewConsumer(cfg config.Kafka, handlers map[string]MessageHandler) (*Consumer, error) {
	// 1. 获取配置信息
	brokers := cfg.GetBrokerList()
	if len(brokers) == 0 {
		return nil, common.ErrKafkaBrokerEmpty
	}
	// 2. 检查是否有处理器
	if len(handlers) == 0 {
		return nil, common.ErrKafkaTopicNotConfig
	}

	readers := make(map[string]*kafka.Reader)

	// 3. 为每个配置的 topic 创建 reader
	for topicKey, topicCfg := range cfg.Topics {
		// 3.1 检查 topic 信息
		if topicCfg.Name == "" {
			continue
		}
		// 3.2 检查消费者组
		if topicCfg.GroupID == "" {
			return nil, common.ErrKafkaTopicNotConfig
		}

		// 3.3 检查是否有对应的 handler
		if _, exists := handlers[topicKey]; !exists {
			return nil, common.ErrKafkaNoValidConsumers
		}
		// 3.4 解析起始偏移量
		startOffset := parseStartOffset(topicCfg.StartOffset)
		// 3.5 创建 reader
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,                                                       // Kafka broker 地址
			Topic:          topicCfg.Name,                                                 // topic 名称
			GroupID:        topicCfg.GroupID,                                              // 消费者组 ID
			GroupBalancers: parseGroupBalancers(cfg.Consumer.GroupBalancer),               // 消费者组均衡器,范围均衡器
			StartOffset:    startOffset,                                                   // 消费起始位置
			MinBytes:       cfg.Consumer.MinBytes,                                         // 最小拉取字节数
			MaxBytes:       cfg.Consumer.MaxBytes,                                         // 最大拉取字节数
			MaxWait:        time.Duration(cfg.Consumer.MaxWait) * time.Millisecond,        // 最大等待时间（毫秒）
			SessionTimeout: time.Duration(cfg.Consumer.SessionTimeout) * time.Millisecond, // 会话超时时间（毫秒）
		})
		readers[topicKey] = reader
	}
	// 如果没有有效 topic，返回错误
	if len(readers) == 0 {
		return nil, common.ErrKafkaNoValidConsumers
	}
	// 4. 返回消费者
	return &Consumer{
		readers:  readers,
		config:   cfg,
		handlers: handlers,
	}, nil
}

// 解析消费者组均衡器
func parseGroupBalancers(strategy string) []kafka.GroupBalancer {
	switch strategy {
	case "range":
		return []kafka.GroupBalancer{kafka.RangeGroupBalancer{}}
	case "round_robin":
		return []kafka.GroupBalancer{kafka.RoundRobinGroupBalancer{}}
	default:
		return []kafka.GroupBalancer{
			kafka.RangeGroupBalancer{},
			kafka.RoundRobinGroupBalancer{},
		}
	}
}

// Start 启动消费者（阻塞）
// 会为每个 topic 启动一个 goroutine 消费消息
func (c *Consumer) Start(ctx context.Context) error {

	c.mu.Lock()
	// 1. 检查消费者是否正在运行
	if c.running {
		c.mu.Unlock()
		// 已经启动，返回报错
		return common.ErrKafkaConsumerRunning
	}
	// 2. 设置消费者为运行中
	c.running = true
	c.mu.Unlock()

	// 3. 创建全局上下文取消函数
	ctx, cancel := context.WithCancel(ctx)
	// 把 cancel 函数绑定到 Consumer 结构体，Close 方法可以调用它统一关停
	c.cancel = cancel

	// 4. 为每个 reader 启动一个消费 goroutine
	for topicKey, reader := range c.readers {
		// 每启动 1 个消费协程，给等待组计数 + 1
		c.wg.Add(1)
		go c.consumeTopic(ctx, topicKey, reader)
	}

	// 5. 等待所有消费 goroutine 退出
	// 等待所有消费者组退出，确保所有消息都被处理完成
	c.wg.Wait()
	return nil
}

// 消费单个 topic 的消息
func (c *Consumer) consumeTopic(ctx context.Context, topicKey string, reader *kafka.Reader) {
	// 1. 确保在退出时关闭 reader，释放资源
	// wg 计数 - 1，标记当前Topic消费协程结束
	defer c.wg.Done()
	// 退出时关闭当前Topic专属Reader，释放TCP连接、缓冲区等资源
	defer reader.Close()

	// 2. 检查是否有对应的 handler
	handler, exists := c.handlers[topicKey]
	if !exists {
		return
	}

	// 3. 从配置中读取批量大小和超时时间
	batchSize := c.config.GetTopicBatchSize(topicKey)
	batchTimeout := time.Duration(c.config.GetTopicBatchTimeout(topicKey)) * time.Millisecond
	fetchTimeout := time.Duration(c.config.GetTopicFetchTimeout(topicKey)) * time.Millisecond
	maxRetries := c.config.GetTopicMaxRetries(topicKey)
	retryWait := time.Duration(c.config.GetTopicRetryWait(topicKey)) * time.Millisecond
	commitMaxRetries := c.config.GetCommitRetries(topicKey)
	commitWait := time.Duration(c.config.GetCommitWait(topicKey)) * time.Millisecond
	// 4. 批量消费模式
	batch := make([]kafka.Message, 0, batchSize)
	ticker := time.NewTicker(batchTimeout) // 超时时间定时器，如果超过超时时间，处理当前批量消息
	defer ticker.Stop()

	//5. 循环拉取消息
	for {
		select {
		case <-ctx.Done(): // 5.1 监听上下文取消信号
			if len(batch) > 0 {
				log.Printf("仍然有%d条消息未处理, topic: %s", len(batch), topicKey)
			}
			return
		case <-ticker.C: // 5.2 监听超时时间定时器信号
			if len(batch) > 0 {
				// 处理当前批量消息
				c.flushBatch(ctx, topicKey, batch, handler, reader, maxRetries, retryWait, commitMaxRetries, commitWait)
				batch = batch[:0] // 清空批量消息
			}
		default: // 5.3 监听默认信号

			// 创建新的上下文，用于拉取单条消息
			newCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
			// 拉取消息
			msg, err := reader.FetchMessage(newCtx)
			// 取消上下文，释放资源
			defer func() {
				cancel()
			}()
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				// 只有网络错误、连接失败才打印报错
				log.Printf("拉取消息真正异常, topic: %s, err: %v", topicKey, err)
				continue
			}
			//  将消息加入批量缓存
			batch = append(batch, msg)

			//  如果批量缓存达到批量大小，立即处理
			if len(batch) >= batchSize {
				c.flushBatch(ctx, topicKey, batch, handler, reader, maxRetries, retryWait, commitMaxRetries, commitWait)
				batch = make([]kafka.Message, 0, batchSize)
				ticker.Reset(batchTimeout) // 重置定时器
			}

		}

	}

}

// 批量消费消息，按照 partition 分组处理，每个 partition 内按 offset 排序处理
func (c *Consumer) flushBatch(ctx context.Context, topicKey string, batch []kafka.Message, handler MessageHandler, reader *kafka.Reader, maxRetries int, retryWait time.Duration, commitMaxRetries int, commitWait time.Duration) {
	// 1. 如果批次为空，直接返回
	if len(batch) == 0 {
		return
	}

	// 2. 记录每个 partition 的最新 offset
	lastOffsets := make(map[int]int64)

	// 3. 逐条处理消息（按 batch 原始顺序）
	for _, msg := range batch {
		// 3.1 处理消息（含重试）
		err := c.handleMessageWithRetry(ctx, topicKey, msg, handler, maxRetries, retryWait)
		if err != nil {
			// 3.2 重试失败，发送到死信队列
			c.sendToDeadLetter(ctx, topicKey, msg, err)
			// 继续处理下一条，不阻塞
		}
		// 3.3 记录该 partition 的最新 offset（无论成功还是失败）
		lastOffsets[msg.Partition] = msg.Offset
	}

	// 4. 如果没有处理任何消息，直接返回
	if len(lastOffsets) == 0 {
		return
	}

	// 5. 构建提交消息列表
	commitMsgs := make([]kafka.Message, 0, len(lastOffsets))
	for partition, offset := range lastOffsets {
		commitMsgs = append(commitMsgs, kafka.Message{
			Topic:     topicKey,
			Partition: partition,
			Offset:    offset,
		})
	}

	// 6. 提交所有 partition 的 offset（含重试）
	for attempt := 0; attempt <= commitMaxRetries; attempt++ {
		// 如果超出最大重试次数，记录日志并返回
		if attempt == commitMaxRetries {
			log.Printf("批量提交最终失败, topic: %s, 消息可能会被重新消费", topicKey)
			return
		}
		// 提交 offset
		err := reader.CommitMessages(ctx, commitMsgs...)
		if err == nil {
			log.Printf("批量提交成功, topic: %s, count: %d, partitions: %d",
				topicKey, len(batch), len(lastOffsets))
			return
		}
		log.Printf("批量提交 offset 失败, topic: %s, attempt: %d/%d, err: %v",
			topicKey, attempt, commitMaxRetries, err)

		if err := waitForRetry(ctx, time.Duration(attempt)*commitWait); err != nil {
			return
		}
	}
}

// 处理单条消息并在失败时按退避间隔重试，返回最后一次错误
func (c *Consumer) handleMessageWithRetry(ctx context.Context, topicKey string, msg kafka.Message, handler MessageHandler, maxRetries int, retryIntervalBase time.Duration) error {
	var lastErr error

	// 1. 尝试处理消息（最多 maxRetries+1 次：首次 + maxRetries 次重试）
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 1.1 调用业务处理器
		lastErr = handler(ctx, string(msg.Key), msg.Value)
		if lastErr == nil {
			return nil
		}

		// 1.2 检查上下文是否取消
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 1.3 记录失败日志
		log.Printf("消息处理失败, topic: %s, partition: %d, offset: %d, attempt: %d/%d, err: %v",
			topicKey, msg.Partition, msg.Offset, attempt+1, maxRetries+1, lastErr)

		// 1.4 如果不是最后一次尝试，等待后重试
		if attempt < maxRetries {
			if err := waitForRetry(ctx, time.Duration(attempt+1)*retryIntervalBase); err != nil {
				return err
			}
		}
	}

	// 2. 所有重试都失败，返回最后一次错误
	return lastErr
}

// 发送消息到死信队列
func (c *Consumer) sendToDeadLetter(ctx context.Context, topicKey string, msg kafka.Message, err error) {
	// 1. 记录日志
	log.Printf("消息进入死信队列, topic: %s, partition: %d, offset: %d, key: %q, err: %v",
		topicKey, msg.Partition, msg.Offset, string(msg.Key), err)

}

// 等待重试，支持上下文取消
func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// 关闭消费者
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 1. 检查消费者是否正在运行
	if !c.running {
		// 没有运行，直接返回
		return nil
	}

	// 2. 取消所有消费 goroutine
	if c.cancel != nil {
		c.cancel()
	}

	// 3. 等待所有 goroutine 退出
	c.wg.Wait()

	// 4. 关闭所有 reader
	var errs []error
	for topicKey, reader := range c.readers {
		if err := reader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭消费者 %s 失败: %w", topicKey, err))
		}
	}

	// 5. 设置消费者为未运行
	c.running = false

	// 如果有错误，返回错误
	if len(errs) > 0 {
		return fmt.Errorf("关闭消费者失败: %v", errs)
	}
	return nil
}

// 检查消费者是否正在运行
func (c *Consumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// 解析起始偏移量
func parseStartOffset(offset string) int64 {
	switch offset {
	case "oldest", "earliest":
		return kafka.FirstOffset
	case "newest", "latest":
		return kafka.LastOffset
	default:
		return kafka.FirstOffset
	}
}
