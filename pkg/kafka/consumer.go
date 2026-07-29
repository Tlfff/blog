package kafka

import (
	"blog/config"
	"blog/internal/common"
	"context"
	"fmt"
	"log"
	"sort"
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
// 返回 error 表示处理失败，消息将不会被提交（等待重试）
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
			StartOffset:    startOffset,                                                   // 消费起始位置：latest
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

// 消费 topic 的消息
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

	// 4. 批量消费模式
	batch := make([]kafka.Message, 0, batchSize)
	ticker := time.NewTicker(batchTimeout) // 超时时间定时器，如果超过超时时间，处理当前批量消息
	defer ticker.Stop()

	// 5. 循环拉取消费消息
	for {
		select {
		// 5.1 监听上下文取消信号
		case <-ctx.Done():
			// 退出前处理剩余消息
			if len(batch) > 0 {
				if err := c.flushBatch(ctx, topicKey, batch, handler, reader); err != nil {
					log.Printf("退出前处理 topic %s 剩余消息失败: %v", topicKey, err)
				}
			}
			return
		// 5.2 监听批量超时
		case <-ticker.C:
			if len(batch) > 0 {
				if err := c.flushBatch(ctx, topicKey, batch, handler, reader); err != nil {
					// 提交失败时不能继续拉取后续消息，否则后续 offset 可能越过未提交消息。
					log.Printf("topic %s 批量消费异常，停止当前 topic 消费: %v", topicKey, err)
					return
				}
				// 重置批量队列
				batch = make([]kafka.Message, 0, batchSize)
			}
		default:
			// 5.3 阻塞拉取单条Kafka消息，可被ctx中断，拉取当前 Reader 分配分区的一条消息
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				// 若为上下文取消导致的报错，正常退出，退出前处理剩余消息
				if ctx.Err() != nil {
					if len(batch) > 0 {
						if err := c.flushBatch(ctx, topicKey, batch, handler, reader); err != nil {
							log.Printf("退出前处理 topic %s 剩余消息失败: %v", topicKey, err)
						}
					}
					return
				}
				// 网络/集群临时故障，不退出，继续循环重试拉取
				continue
			}
			// 5.3  累加消息到批量处理队列
			batch = append(batch, msg)

			// 5.4 批量处理
			// 如果批量队列大小超过配置的批量大小，处理当前批量消息
			if len(batch) >= batchSize {
				if err := c.flushBatch(ctx, topicKey, batch, handler, reader); err != nil {
					// 提交失败时保守停止，避免消费并提交后续消息。
					log.Printf("topic %s 批量消费异常，停止当前 topic 消费: %v", topicKey, err)
					return
				}
				batch = make([]kafka.Message, 0, batchSize)
				ticker.Reset(batchTimeout) // 重置定时器
			}
		}
	}
}

// flushBatch 处理一个 topic 的批次，并只提交每个 partition 的最后一条安全消息。
func (c *Consumer) flushBatch(ctx context.Context, topicKey string, batch []kafka.Message, handler MessageHandler, reader *kafka.Reader) error {
	if len(batch) == 0 {
		return nil
	}

	commitMessages, err := c.processBatch(ctx, topicKey, batch, handler)
	if err != nil {
		return err
	}
	if len(commitMessages) == 0 {
		return nil
	}

	for attempt := 1; attempt <= maxCommitAttempts; attempt++ {
		if err := reader.CommitMessages(ctx, commitMessages...); err == nil {
			log.Printf("批量处理成功, topic: %s, count: %d, partitions: %d", topicKey, len(batch), len(commitMessages))
			return nil
		} else {
			log.Printf("批量提交 offset 失败, topic: %s, attempt: %d/%d, err: %v", topicKey, attempt, maxCommitAttempts, err)
			if attempt == maxCommitAttempts {
				return err
			}
		}

		if err := waitForRetry(ctx, time.Duration(attempt)*100*time.Millisecond); err != nil {
			return err
		}
	}

	return nil
}

// processBatch 按 partition 分组处理消息，并返回每个 partition 可安全提交的最后一条消息。
// 消息首次处理失败后最多再重试 3 次，仍失败则记录并丢弃，然后继续处理同批次后续消息。
func (c *Consumer) processBatch(ctx context.Context, topicKey string, batch []kafka.Message, handler MessageHandler) ([]kafka.Message, error) {
	grouped := make(map[int][]kafka.Message)
	for _, msg := range batch {
		grouped[msg.Partition] = append(grouped[msg.Partition], msg)
	}

	commitMessages := make([]kafka.Message, 0, len(grouped))
	for partition, messages := range grouped {
		sort.SliceStable(messages, func(i, j int) bool {
			return messages[i].Offset < messages[j].Offset
		})

		var lastSafe kafka.Message
		for _, msg := range messages {
			if err := c.handleMessageWithRetry(ctx, topicKey, msg, handler); err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				// 当前策略：达到最大重试次数后丢弃消息，并将其视为已处理，
				// 这样该 partition 才能继续向后推进。
				log.Printf("消息重试 %d 次后丢弃, topic: %s, partition: %d, offset: %d, key: %q, err: %v", maxMessageRetries, topicKey, partition, msg.Offset, string(msg.Key), err)
			}
			lastSafe = msg
		}

		commitMessages = append(commitMessages, lastSafe)
	}

	return commitMessages, nil
}

func (c *Consumer) handleMessageWithRetry(ctx context.Context, topicKey string, msg kafka.Message, handler MessageHandler) error {
	var lastErr error
	for attempt := 0; attempt <= maxMessageRetries; attempt++ {
		lastErr = handler(ctx, string(msg.Key), msg.Value)
		if lastErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		log.Printf("消息处理失败，将重试, topic: %s, partition: %d, offset: %d, attempt: %d/%d, err: %v", topicKey, msg.Partition, msg.Offset, attempt+1, maxMessageRetries+1, lastErr)
		if attempt < maxMessageRetries {
			if err := waitForRetry(ctx, time.Duration(attempt+1)*100*time.Millisecond); err != nil {
				return err
			}
		}
	}

	return lastErr
}

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
