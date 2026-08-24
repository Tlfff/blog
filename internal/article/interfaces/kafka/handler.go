// Package kafka 提供 Article 上下文的浏览历史 Kafka Adapter。
package kafka

import (
	"blog/internal/platform/kafka"
	"blog/internal/platform/kafka/message"
	"context"
	"encoding/json"
	"log"
	"time"
)

// Consumer 消费浏览历史消息。
type Consumer interface {
	// CreateViewHistory 记录浏览历史并增加文章浏览量。
	CreateViewHistory(ctx context.Context, userID, articleID uint64, timestamp time.Time) error
}

// NewHandler 创建浏览历史 Kafka 消息处理器。
func NewHandler(consumer Consumer) kafka.MessageHandler {
	return func(ctx context.Context, key string, value []byte) error {
		// 1. 解析基线 ViewHistoryMsg JSON
		var msg message.ViewHistoryMsg
		if err := json.Unmarshal(value, &msg); err != nil {
			log.Printf("[MQ] 解析浏览历史消息失败: %v", err)
			return err
		}

		// 2. 调用 Article Application 处理浏览历史
		if err := consumer.CreateViewHistory(ctx, msg.UserID, msg.ArticleID, msg.CreatedTime); err != nil {
			log.Printf("[MQ] 处理浏览历史失败: %v", err)
			return err
		}
		log.Printf("[MQ] 浏览历史处理成功, user: %d, article: %d", msg.UserID, msg.ArticleID)
		return nil
	}
}
