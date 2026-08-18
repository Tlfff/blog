package kafka

import (
	"blog/internal/infrastructure/config"
	"blog/internal/common"
	"blog/internal/message"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
)

// Producer 是 Kafka 生产者，为每个 topic 维护独立的 writer。
type Producer struct {
	writers map[string]*kafka.Writer // 各 topic 对应的 writer，key 为 topic 配置名
	config  config.Kafka // Kafka 配置
}

// 创建 Kafka 生产者。
func NewProducer(cfg config.Kafka) (*Producer, error) {
	// 1. 获取配置信息
	// 1.1 验证 broker 地址列表是否为空
	brokers := cfg.GetBrokerList()
	if len(brokers) == 0 {
		return nil, common.ErrKafkaBrokerEmpty
	}

	// 1.2 解析压缩类型
	compression := parseCompression(cfg.Producer.CompressionType)

	// 1.3 解析确认级别
	requiredAcks := parseRequiredAcks(cfg.Producer.Acks)

	// 2. 为每个 topic 创建 writer
	writers := make(map[string]*kafka.Writer)
	for topicKey, topicCfg := range cfg.Topics {

		// 2.1 验证 topic 名称是否为空
		if topicCfg.Name == "" {
			return nil, common.ErrKafkaTopicNotConfig
		}
		balancer := getBalancer(topicKey)
		// 2.2 创建 writer
		writer := &kafka.Writer{
			Addr:         kafka.TCP(brokers...),                                       // 配置 broker 地址
			Topic:        topicCfg.Name,                                               // 配置 topic 名称
			Balancer:     balancer,                                                    // 配置负载均衡器
			RequiredAcks: requiredAcks,                                                // 配置确认级别
			Compression:  compression,                                                 // 配置压缩类型
			MaxAttempts:  cfg.Producer.MaxRetries,                                     // 配置最大重试次数
			BatchSize:    cfg.Producer.BatchSize,                                      // 配置批量大小
			BatchBytes:   int64(cfg.Producer.BatchSizeBytes),                          // 配置批量大小（字节）
			BatchTimeout: time.Duration(cfg.Producer.BatchTimeout) * time.Millisecond, // 配置批量超时时间
			Async:        false,                                                       // 同步发送，可以获取ack
		}
		writers[topicKey] = writer
	}
	// 2.3 验证是否有有效 topic 配置
	if len(writers) == 0 {
		return nil, common.ErrKafkaTopicNotConfig
	}

	// 3. 返回生产者实例
	return &Producer{
		writers: writers,
		config:  cfg,
	}, nil
}
// 按 topic 选择分区负载均衡策略
func getBalancer(topicKey string) kafka.Balancer {
	switch topicKey {
	case "notification":
		return &kafka.RoundRobin{} // 通知消息使用轮询分区，提高吞吐量
	case "view_history":
		return &kafka.Hash{} // 浏览历史消息使用 Key Hash 分区，保证同一用户的消息有序
	default:
		return &kafka.LeastBytes{} // 其他消息使用 Least Bytes 分区，确保消息均匀分布
	}
}

// 发送浏览历史消息（使用 UserID 作为 Key，保证同一用户的消息有序）
func (p *Producer) SendViewHistory(ctx context.Context, msg *message.ViewHistoryMsg) error {
	key := fmt.Sprintf("%d", msg.UserID)
	return p.sendMessage(ctx, "view_history", []byte(key), msg)
}

// 发送通知消息（无 Key，使用轮询分区）
func (p *Producer) SendNotificationAsync(msg *message.NotificationMsg) {
	go func() {
		// 1. 捕获异常，防止 goroutine 崩溃影响主流程
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[Kafka] 异步发送通知消息 panic: %v", err)
			}
		}()

		// 2. 创建新上下文，设置发送超时时间
		// 注意：不能使用请求的 ctx，请求结束后它会被取消，导致发送失败
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 3. 发送消息，失败只打日志，通知类消息允许丢失
		if err := p.sendMessage(ctx, "notification", nil, msg); err != nil {
			log.Printf("[Kafka] 异步发送通知消息失败: %v", err)
		}
	}()
}

// 通用发送方法
// key 为 nil 时，Kafka 使用轮询分区（Round-Robin）
// key 不为 nil 时，Kafka 使用 Key Hash 分区（相同 Key 分配到同一分区）
func (p *Producer) sendMessage(ctx context.Context, topicKey string, key []byte, value interface{}) error {
	// 1. 按 topicKey 取出对应的 writer，未配置则报错
	// 1. 获取 writer
	writer, exists := p.writers[topicKey]
	if !exists {
		return common.ErrKafkaTopicNotConfig
	}
	// 2. 序列化消息
	data, err := json.Marshal(value)
	if err != nil {
		return common.ErrKafkaSerializeFailed
	}
	// 3. 构建消息
	msg := kafka.Message{
		Key:   key, // nil = 轮询分区，非 nil = Key Hash 分区
		Value: data,
	}

	// 4. 同步发送，获取 ACK 确认
	return writer.WriteMessages(ctx, msg)
}

// 关闭所有 writer
func (p *Producer) Close() error {
	// 1. 逐个关闭 writer，收集所有关闭错误
	var errs []error // 收集关闭过程中的错误
	// 1. 关闭所有 writer
	for topicKey, writer := range p.writers {
		if err := writer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 topic %s 失败，错误: %w", topicKey, err))
		}
	}
	// 2. 检查是否有错误
	if len(errs) > 0 {
		return common.ErrKafkaCloseFailed
	}
	return nil
}

// 辅助函数：解析压缩类型
func parseCompression(compressionType string) compress.Compression {
	// 1. 按配置名匹配压缩算法，未匹配时兜底为 snappy
	switch compressionType {
	case "none":
		return compress.None
	case "gzip":
		return compress.Gzip
	case "snappy":
		return compress.Snappy
	case "lz4":
		return compress.Lz4
	case "zstd":
		return compress.Zstd
	default:
		return compress.Snappy // 默认使用 snappy
	}
}

// 辅助函数：解析确认级别
func parseRequiredAcks(acks string) kafka.RequiredAcks {
	// 1. 按配置名匹配确认级别，未匹配时兜底为 one
	switch acks {
	case "none":
		return kafka.RequireNone
	case "one":
		return kafka.RequireOne
	case "all":
		return kafka.RequireAll
	default:
		return kafka.RequireOne // 默认使用 one
	}
}
