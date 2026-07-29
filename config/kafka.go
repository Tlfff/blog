package config

import "strings"

type Kafka struct {
	Brokers  string                 `yaml:"brokers"`  //kafka broker 地址
	Topics   map[string]TopicConfig `yaml:"topics"`   // 所有 topic 配置
	Producer KafkaProducer          `yaml:"producer"` // 生产者配置
	Consumer KafkaConsumer          `yaml:"consumer"` // 消费者配置
}

// kafka 主题名称配置
type TopicConfig struct {
	Name         string `yaml:"name"`             // topic 名称
	GroupID      string `yaml:"group_id"`         // 消费者组 ID
	StartOffset  string `yaml:"start_offset"`     // 消费起始位置：oldest/latest
	BatchSize    int    `yaml:"batch_size"`       // 批量消费条数，默认 50
	BatchTimeout int    `yaml:"batch_timeout_ms"` // 批量超时时间（毫秒），默认 1000
}

// kafka 生产者配置
type KafkaProducer struct {
	Acks            string `yaml:"acks"`             // 确认数: none/one/all
	BatchSize       int    `yaml:"batch_size"`       // 批量消息条数
	BatchSizeBytes  int    `yaml:"batch_size_bytes"` // 批量字节数
	BatchTimeout    int    `yaml:"batch_timeout_ms"` // 批量超时时间（毫秒）
	MaxRetries      int    `yaml:"max_retries"`      // 最大重试次数
	CompressionType string `yaml:"compression_type"` // 压缩类型: none/gzip/snappy/lz4/zstd
}

// kafka 消费者配置
type KafkaConsumer struct {
	MaxBytes       int `yaml:"max_bytes"`          // 最大拉取字节数
	MinBytes       int `yaml:"min_bytes"`          // 最小拉取字节数
	MaxWait        int `yaml:"max_wait_ms"`        // 最大等待时间（毫秒）
	SessionTimeout int `yaml:"session_timeout_ms"` // 会话超时时间（毫秒）
	GroupBalancer   string `yaml:"group_balancer"`
}

// 获取 kafka broker 地址列表
func (k *Kafka) GetBrokerList() []string {
	if k.Brokers == "" {
		return nil
	}
	brokers := strings.Split(k.Brokers, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}
	return brokers
}

// 获取 kafka 主题名称列表
func (k *Kafka) GetTopicNames() []string {
	names := make([]string, 0, len(k.Topics))
	for _, topic := range k.Topics {
		if topic.Name != "" {
			names = append(names, topic.Name)
		}
	}
	return names
}

// 获取 topic 的批量消费大小，如果未配置则使用默认值
func (k *Kafka) GetTopicBatchSize(topicKey string) int {
	if topic, exists := k.Topics[topicKey]; exists && topic.BatchSize > 0 {
		return topic.BatchSize
	}
	return 50 // 默认 50 条
}

// 获取 topic 的批量超时时间，如果未配置则使用默认值
func (k *Kafka) GetTopicBatchTimeout(topicKey string) int {
	if topic, exists := k.Topics[topicKey]; exists && topic.BatchTimeout > 0 {
		return topic.BatchTimeout
	}
	return 1000 // 默认 1000ms
}
