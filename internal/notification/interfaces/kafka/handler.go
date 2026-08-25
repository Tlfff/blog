// Package kafka 提供 Notification 上下文的 Kafka 协议适配器。
package kafka

import (
	notificationdomain "blog/internal/notification/domain"
	pkgkafka "blog/internal/platform/kafka"
	"blog/internal/platform/kafka/message"
	"context"
	"encoding/json"
	"log"
)

// Consumer 消费通知业务事件。
type Consumer interface {
	// CreateLikeNotification 处理当前基线已接线的文章点赞通知。
	CreateLikeNotification(ctx context.Context, event notificationdomain.NotificationEvent) error
}

// NewHandler 创建 Notification 上下文的 Kafka 消息处理器。
func NewHandler(consumer Consumer) pkgkafka.MessageHandler {
	return func(ctx context.Context, key string, value []byte) error {
		// 1. 解析基线 NotificationMsg JSON
		var msg message.NotificationMsg
		if err := json.Unmarshal(value, &msg); err != nil {
			log.Printf("[MQ] 解析通知消息失败: %v", err)
			return err
		}

		// 2. 调用通知应用服务处理消息
		return consumer.CreateLikeNotification(ctx, notificationdomain.NotificationEvent{
			NotifyType:  msg.NotifyType,
			SenderID:    msg.SenderID,
			TargetID:    msg.TargetID,
			CreatedTime: msg.CreatedTime,
		})
	}
}
