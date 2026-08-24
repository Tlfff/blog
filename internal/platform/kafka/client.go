package kafka

import (
	"blog/internal/platform/config"
	apperrors "blog/internal/shared/apperrors"
	"context"
	"errors"
	"log"
	"sync"
)

// 管理生产者和消费者的生命周期
type Client struct {
	mu       sync.RWMutex // 保护生产者和消费者的访问
	producer *Producer    // Kafka 生产者
	consumer *Consumer    // Kafka 消费者
	config   config.Kafka // Kafka 配置
	closed   bool         // 当前客户端是否已关闭
}

// NewClient 创建 Kafka 客户端
// 注意：只初始化 Producer，Consumer 需要单独调用 InitConsumer
func NewClient(cfg config.Kafka) (*Client, error) {
	// 1. 获取配置信息
	if len(cfg.GetBrokerList()) == 0 {
		return nil, apperrors.ErrKafkaBrokerEmpty
	}
	// 2. 初始化客户端
	client := &Client{
		config: cfg,
	}

	// 3. 初始化 Producer
	producer, err := NewProducer(cfg)
	if err != nil {
		return nil, apperrors.ErrKafkaInitFailed
	}
	client.producer = producer
	// // 4. 预热连接
	// if err := producer.Ping(context.Background()); err != nil {
	// 	return nil, apperrors.ErrKafkaPingFailed
	// }
	// 5. 返回客户端
	return client, nil
}

// GetProducer 获取 Kafka Producer。
//
// 如果客户端已关闭，返回 nil
func (c *Client) GetProducer() *Producer {
	// 1. 检查客户端是否已关闭
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return nil
	}
	// 2. 返回生产者
	return c.producer
}

// InitConsumer 初始化 Kafka Consumer。
func (c *Client) InitConsumer(handlers map[string]MessageHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 1. 检查客户端是否已关闭
	if c.closed {
		return apperrors.ErrKafkaClientClosed
	}
	// 2. 检查消费者是否已初始化
	if c.consumer != nil {
		return nil // 已经初始化
	}
	// 3. 检查是否有有效消费者配置
	if len(handlers) == 0 {
		return apperrors.ErrKafkaNoValidConsumers
	}
	// 4. 初始化消费者
	consumer, err := NewConsumer(c.config, handlers)
	if err != nil {
		return apperrors.ErrKafkaInitFailed
	}
	c.consumer = consumer
	return nil
}

// StartConsumer 启动 Kafka Consumer。
// 需要先调用 InitConsumer
func (c *Client) StartConsumer(ctx context.Context) error {

	c.mu.RLock()
	consumer := c.consumer
	closed := c.closed
	c.mu.RUnlock()
	// 1. 检查客户端是否已关闭
	if closed {
		return apperrors.ErrKafkaClientClosed
	}
	// 2. 检查消费者是否已初始化
	if consumer == nil {
		return apperrors.ErrKafkaConsumerRunning
	}

	return consumer.Start(ctx)
}

// StopConsumer 停止 Kafka Consumer。
func (c *Client) StopConsumer() error {
	c.mu.RLock()
	consumer := c.consumer
	c.mu.RUnlock()
	// 1. 检查消费者是否已初始化
	if consumer == nil {
		return nil
	}
	// 2. 关闭消费者
	return consumer.Close()
}

// IsConsumerReady 检查 Consumer 是否已初始化。
func (c *Client) IsConsumerReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consumer != nil && !c.closed
}

// IsConsumerRunning 检查 Consumer 是否正在运行。
func (c *Client) IsConsumerRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// 1. 检查消费者是否已初始化
	if c.consumer == nil || c.closed {
		return false
	}
	// 2. 检查消费者是否正在运行
	return c.consumer.IsRunning()
}

// Close 关闭 Kafka 客户端并释放资源。
func (c *Client) Close() error {
	// 1. 检查客户端是否已关闭
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	var errs []error

	// 2. 关闭 Producer
	if c.producer != nil {
		if err := c.producer.Close(); err != nil {
			log.Printf("关闭生产者失败: %v", err)
			errs = append(errs, apperrors.ErrKafkaCloseFailed)
		}
		c.producer = nil
	}

	// 3. 关闭 Consumer
	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			log.Printf("关闭消费者失败: %v", err)
			errs = append(errs, apperrors.ErrKafkaCloseFailed)
		}
		c.consumer = nil
	}
	// 4. 返回结果
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
