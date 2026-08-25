package config

import "strings"

// Kafka 表示 Kafka 生产、消费和死信配置。
type Kafka struct {
	Brokers    string                 `yaml:"brokers"`     // Kafka Broker 地址
	Topics     map[string]TopicConfig `yaml:"topics"`      // 所有 topic 配置
	Producer   KafkaProducer          `yaml:"producer"`    // 生产者配置
	Consumer   KafkaConsumer          `yaml:"consumer"`    // 消费者配置
	DeadLetter DeadLetterConfig       `yaml:"dead_letter"` // 死信队列配置
}

// TopicConfig 表示单个 Kafka Topic 配置。
type TopicConfig struct {
	Name         string `yaml:"name"`             // topic 名称
	GroupID      string `yaml:"group_id"`         // 消费者组 ID
	StartOffset  string `yaml:"start_offset"`     // 消费起始位置：oldest/latest
	BatchSize    int    `yaml:"batch_size"`       // 批量消费条数，默认 50
	BatchTimeout int    `yaml:"batch_timeout_ms"` // 批量超时时间（毫秒），默认 1000
	FetchTimeout int    `yaml:"fetch_timeout_ms"` // 拉取单条消息的超时时间（毫秒），默认 300
	MaxRetries   int    `yaml:"max_retries"`      // 消息处理最大重试次数，默认 3
	RetryWait    int    `yaml:"retry_wait_ms"`    // 重试间隔（毫秒），默认 100
}

// KafkaProducer 表示 Kafka Producer 配置。
type KafkaProducer struct {
	Acks            string `yaml:"acks"`             // 确认数: none/one/all
	BatchSize       int    `yaml:"batch_size"`       // 批量消息条数
	BatchSizeBytes  int    `yaml:"batch_size_bytes"` // 批量字节数
	BatchTimeout    int    `yaml:"batch_timeout_ms"` // 批量超时时间（毫秒）
	MaxRetries      int    `yaml:"max_retries"`      // 最大重试次数
	CompressionType string `yaml:"compression_type"` // 压缩类型: none/gzip/snappy/lz4/zstd
}

// KafkaConsumer 表示 Kafka Consumer 配置。
type KafkaConsumer struct {
	MaxBytes       int    `yaml:"max_bytes"`          // 最大拉取字节数
	MinBytes       int    `yaml:"min_bytes"`          // 最小拉取字节数
	MaxWait        int    `yaml:"max_wait_ms"`        // 最大等待时间（毫秒）
	SessionTimeout int    `yaml:"session_timeout_ms"` // 会话超时时间（毫秒）
	GroupBalancer  string `yaml:"group_balancer"`     // 消费者组分配策略，默认 round_robin
	CommitRetries  int    `yaml:"commit_retries"`     // 提交偏移量重试次数，默认 3 次
	CommitWait     int    `yaml:"commit_wait_ms"`     // 提交偏移量间隔，默认 100 毫秒
}

// DeadLetterConfig 表示 Kafka 死信队列配置。
type DeadLetterConfig struct {
	Topic         string `yaml:"topic"`          // 死信队列 topic 名称
	MaxRetries    int    `yaml:"max_retries"`    // 发送到死信队列的重试次数（网络层面）
	ConsumerGroup string `yaml:"consumer_group"` // 死信队列消费者组 ID
}

// GetBrokerList 获取 Kafka Broker 地址列表。
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

// GetTopicNames 获取 Kafka Topic 名称列表。
func (k *Kafka) GetTopicNames() []string {
	names := make([]string, 0, len(k.Topics))
	for _, topic := range k.Topics {
		if topic.Name != "" {
			names = append(names, topic.Name)
		}
	}
	return names
}

// GetTopicBatchSize 获取 Topic 批量消费大小，未配置时返回默认值。
func (k *Kafka) GetTopicBatchSize(topicKey string) int {
	if topic, exists := k.Topics[topicKey]; exists && topic.BatchSize > 0 {
		return topic.BatchSize
	}
	return 50 // 默认 50 条
}

// GetTopicBatchTimeout 获取 Topic 批量超时时间，未配置时返回默认值。
func (k *Kafka) GetTopicBatchTimeout(topicKey string) int {
	if topic, exists := k.Topics[topicKey]; exists && topic.BatchTimeout > 0 {
		return topic.BatchTimeout
	}
	return 1000 // 默认 1000ms
}

// GetTopicFetchTimeout 获取 Topic 单条消息拉取超时，未配置时返回默认值。
func (k *Kafka) GetTopicFetchTimeout(topicKey string) int {
	if topic, exists := k.Topics[topicKey]; exists && topic.FetchTimeout > 0 {
		return topic.FetchTimeout
	}
	return 300 // 默认 300ms
}

// GetTopicMaxRetries 获取 Topic 最大处理重试次数，未配置时返回默认值。
func (k *Kafka) GetTopicMaxRetries(topicKey string) int {
	if topic, exists := k.Topics[topicKey]; exists && topic.MaxRetries > 0 {
		return topic.MaxRetries
	}
	return 3 // 默认 3 次
}

// GetTopicRetryWait 获取 Topic 处理重试间隔，未配置时返回默认值。
func (k *Kafka) GetTopicRetryWait(topicKey string) int {
	if topic, exists := k.Topics[topicKey]; exists && topic.RetryWait > 0 {
		return topic.RetryWait
	}
	return 100 // 默认 100ms
}

// GetCommitRetries 获取 Offset 提交重试次数，未配置时返回默认值。
func (k *Kafka) GetCommitRetries(topicKey string) int {
	if k.Consumer.CommitRetries > 0 {
		return k.Consumer.CommitRetries
	}
	return 3 // 默认 3 次
}

// GetCommitWait 获取 Offset 提交重试间隔，未配置时返回默认值。
func (k *Kafka) GetCommitWait(topicKey string) int {
	if k.Consumer.CommitWait > 0 {
		return k.Consumer.CommitWait
	}
	return 100 // 默认 100ms
}

// GetDeadLetterTopic 获取死信队列 Topic 名称。
func (k *Kafka) GetDeadLetterTopic() string {
	if k.DeadLetter.Topic != "" {
		return k.DeadLetter.Topic
	}
	return "dead_letter" // 默认死信队列 topic 名称
}

// GetDeadLetterMaxRetries 获取死信消息发送最大重试次数。
func (k *Kafka) GetDeadLetterMaxRetries() int {
	if k.DeadLetter.MaxRetries > 0 {
		return k.DeadLetter.MaxRetries
	}
	return 3 // 默认 3 次
}

// GetDeadLetterConsumerGroup 获取死信队列消费者组 ID。
func (k *Kafka) GetDeadLetterConsumerGroup() string {
	if k.DeadLetter.ConsumerGroup != "" {
		return k.DeadLetter.ConsumerGroup
	}
	return "dead_letter_consumer" // 默认死信队列消费者组 ID
}
