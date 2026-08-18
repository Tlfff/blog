package kafka

import (
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// 死信队列消息结构
// DeadLetterMessage 是死信队列消息载荷，记录原始消息与失败原因。
type DeadLetterMessage struct {
	// 原始消息信息
	SourceTopic string `json:"source_topic"` // 原始 topic 名称（如 "notification"）
	Partition   int    `json:"partition"`    // 原始分区
	Offset      int64  `json:"offset"`       // 原始 offset
	Key         string `json:"key"`          // 原始消息 key
	Value       string `json:"value"`        // 原始消息 value（JSON 字符串）

	// 错误信息
	Error     string    `json:"error"`     // 最后一次处理的错误信息
	Retries   int       `json:"retries"`   // 已经重试的次数（= 配置的最大重试次数）
	Timestamp time.Time `json:"timestamp"` // 进入死信队列的时间

	// 元信息
	ConsumerGroup string `json:"consumer_group"` // 消费者组 ID
}

// 将死信消息序列化为 JSON 字节数组
// 将死信消息序列化为 JSON 字节数组
func (m *DeadLetterMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// 创建死信消息
// 构造一条死信消息，汇总原始消息、错误信息与消费者组
func NewDeadLetterMessage(sourceTopic string, consumerGroup string, msg kafka.Message, err error, retries int) *DeadLetterMessage {
	return &DeadLetterMessage{
		SourceTopic:   sourceTopic,
		Partition:     msg.Partition,
		Offset:        msg.Offset,
		Key:           string(msg.Key),
		Value:         string(msg.Value),
		Error:         err.Error(),
		Retries:       retries,
		Timestamp:     time.Now(),
		ConsumerGroup: consumerGroup,
	}
}
