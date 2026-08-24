package kafka

import (
	"blog/internal/platform/config"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// DeadLetterProducer 死信队列生产者
type DeadLetterProducer struct {
	writer *kafka.Writer // 死信 Topic Writer
	config config.Kafka  // Kafka 配置
	closed bool          // 不需要锁保护，因为 Close 只在程序退出时调用
}

// NewDeadLetterProducer 创建死信队列 Producer。
func NewDeadLetterProducer(cfg config.Kafka) (*DeadLetterProducer, error) {
	// 1. 获取配置
	brokers := cfg.GetBrokerList()
	if len(brokers) == 0 {
		return nil, fmt.Errorf("broker 地址列表为空")
	}

	dlqTopic := cfg.GetDeadLetterTopic()
	if dlqTopic == "" {
		return nil, fmt.Errorf("死信队列 topic 未配置")
	}

	maxRetries := cfg.GetDeadLetterMaxRetries()

	// 2. 创建 Writer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        dlqTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Compression:  parseCompression("snappy"),
		MaxAttempts:  maxRetries,
		BatchSize:    10,
		BatchBytes:   16384,
		BatchTimeout: 200 * time.Millisecond,
		Async:        false,
	}

	// 3. 返回生产者实例
	return &DeadLetterProducer{
		writer: writer,
		config: cfg,
		closed: false,
	}, nil
}

// Send 发送消息到死信队列
func (p *DeadLetterProducer) Send(ctx context.Context, dlMsg *DeadLetterMessage) error {
	if p.closed {
		return fmt.Errorf("死信队列生产者已关闭")
	}

	// 1. 构建 Kafka 消息
	// Value 直接使用原始消息的 Value
	msg := kafka.Message{
		Key:   []byte(dlMsg.Key),   // 保持原始 Key
		Value: []byte(dlMsg.Value), // 保持原始 Value
		Time:  dlMsg.Timestamp,
	}

	// 2. 元信息放在 Headers 中
	msg.Headers = []kafka.Header{
		{Key: "x-dlq-source-topic", Value: []byte(dlMsg.SourceTopic)},
		{Key: "x-dlq-source-partition", Value: []byte(fmt.Sprintf("%d", dlMsg.Partition))},
		{Key: "x-dlq-source-offset", Value: []byte(fmt.Sprintf("%d", dlMsg.Offset))},
		{Key: "x-dlq-error", Value: []byte(dlMsg.Error)},
		{Key: "x-dlq-retries", Value: []byte(fmt.Sprintf("%d", dlMsg.Retries))},
		{Key: "x-dlq-timestamp", Value: []byte(dlMsg.Timestamp.Format(time.RFC3339))},
		{Key: "x-dlq-consumer-group", Value: []byte(dlMsg.ConsumerGroup)},
	}

	// 3. 发送到死信队列
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("发送死信消息失败: %w", err)
	}

	log.Printf("死信消息发送成功, source_topic: %s, partition: %d, offset: %d, dlq_topic: %s",
		dlMsg.SourceTopic, dlMsg.Partition, dlMsg.Offset, p.writer.Topic)

	return nil
}

// SendWithRetry 按配置重试发送死信消息。
func (p *DeadLetterProducer) SendWithRetry(ctx context.Context, dlMsg *DeadLetterMessage, maxRetries int) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := p.Send(ctx, dlMsg)
		if err == nil {
			return nil
		}
		lastErr = err

		log.Printf("发送死信消息失败, attempt: %d/%d, source_topic: %s, offset: %d, err: %v",
			attempt+1, maxRetries+1, dlMsg.SourceTopic, dlMsg.Offset, err)

		if attempt < maxRetries {
			if err := waitForRetry(ctx, time.Duration(attempt+1)*100*time.Millisecond); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("发送死信消息最终失败: %w", lastErr)
}

// Close 关闭死信队列 Producer。
func (p *DeadLetterProducer) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true

	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			return fmt.Errorf("关闭死信队列 Writer 失败: %w", err)
		}
	}

	log.Printf("死信队列生产者已关闭")
	return nil
}

// IsClosed 检查死信队列 Producer 是否已关闭。
func (p *DeadLetterProducer) IsClosed() bool {
	return p.closed
}
