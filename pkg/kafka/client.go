package kafka

import (
	"blog/internal/infrastructure/config"
	"blog/internal/common"
	"context"
	"errors"
	"log"
	"sync"
)

// 管理生产者和消费者的生命周期
// Client 是 Kafka 客户端，统一管理生产者与消费者的生命周期。
type Client struct {
	mu       sync.RWMutex // 保护生产者和消费者的访问
	producer *Producer    // Kafka 生产者
	consumer *Consumer    // Kafka 消费者
	config   config.Kafka // Kafka 配置
	closed   bool         // 当前客户端是否已关闭
}

// NewClient 创建 Kafka 客户端
// 注意：只初始化 Producer，Consumer 需要单独调用 InitConsumer
// 创建 Kafka 客户端，只初始化 Producer；Consumer 需另行调用 InitConsumer
func NewClient(cfg config.Kafka) (*Client, error) {
	// 1. 获取配置信息
	if len(cfg.GetBrokerList()) == 0 {
		return nil, common.ErrKafkaBrokerEmpty
	}
	// 2. 初始化客户端
	client := &Client{
		config: cfg,
	}

	// 3. 初始化 Producer
	producer, err := NewProducer(cfg)
	if err != nil {
		return nil, common.ErrKafkaInitFailed
	}
	client.producer = producer
	// // 4. 预热连接
	// if err := producer.Ping(context.Background()); err != nil {
	// 	return nil, common.ErrKafkaPingFailed
	// }
	// 5. 返回客户端
	return client, nil
}

//	获取生产者
//
// 如果客户端已关闭，返回 nil
// 获取生产者，客户端已关闭时返回 nil
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

// 初始化消费者
// 初始化消费者，重复调用不会重建已有消费者
func (c *Client) InitConsumer(handlers map[string]MessageHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 1. 检查客户端是否已关闭
	if c.closed {
		return common.ErrKafkaClientClosed
	}
	// 2. 检查消费者是否已初始化
	if c.consumer != nil {
		return nil // 已经初始化
	}
	// 3. 检查是否有有效消费者配置
	if len(handlers) == 0 {
		return common.ErrKafkaNoValidConsumers
	}
	// 4. 初始化消费者
	consumer, err := NewConsumer(c.config, handlers)
	if err != nil {
		return common.ErrKafkaInitFailed
	}
	c.consumer = consumer
	return nil
}

// 启动消费者
// 需要先调用 InitConsumer
// 启动消费者，需先调用 InitConsumer
func (c *Client) StartConsumer(ctx context.Context) error {

	c.mu.RLock()
	consumer := c.consumer
	closed := c.closed
	c.mu.RUnlock()
	// 1. 检查客户端是否已关闭
	if closed {
		return common.ErrKafkaClientClosed
	}
	// 2. 检查消费者是否已初始化
	if consumer == nil {
		return common.ErrKafkaConsumerRunning
	}

	return consumer.Start(ctx)
}

// 关闭消费者
// 关闭消费者，未初始化时直接返回 nil
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

// 检查消费者是否已初始化
// 判断消费者是否已初始化且客户端未关闭
func (c *Client) IsConsumerReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consumer != nil && !c.closed
}

// 检查消费者是否正在运行
// 判断消费者是否正在运行
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

// 关闭客户端，释放所有资源
// 关闭客户端并释放生产者与消费者资源
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
			errs = append(errs, common.ErrKafkaCloseFailed)
		}
		c.producer = nil
	}

	// 3. 关闭 Consumer
	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			log.Printf("关闭消费者失败: %v", err)
			errs = append(errs, common.ErrKafkaCloseFailed)
		}
		c.consumer = nil
	}
	// 4. 返回结果
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
