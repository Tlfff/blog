package mq

import (
	domainnotification "blog/internal/notification/domain"
	message "blog/shared/contracts/events/messages"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"blog/pkg/kafka"
)

// 创建通知消息处理器
func NewNotificationHandler(consumer NotificationConsumer) kafka.MessageHandler {
	return func(ctx context.Context, key string, value []byte) error {
		// 1. 解析消息
		var msg message.NotificationMsg
		if err := json.Unmarshal(value, &msg); err != nil {
			log.Printf("[MQ] 解析通知消息失败: %v", err)
			return err
		}
		if msg.EventID == "" || msg.Version != 1 {
			log.Printf("[MQ] 拒绝无效通知事件: event_id=%q version=%d", msg.EventID, msg.Version)
			return fmt.Errorf("unsupported notification event: id=%q version=%d", msg.EventID, msg.Version)
		}
		return consumer.CreateLikeNotification(ctx, domainnotification.NotificationEvent{
			EventID:     msg.EventID,
			Version:     msg.Version,
			NotifyType:  msg.NotifyType,
			SenderID:    msg.SenderID,
			TargetID:    msg.TargetID,
			CreatedTime: msg.CreatedTime,
		})
	}
}
