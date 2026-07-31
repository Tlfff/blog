package mq

import (
	"blog/internal/message"
	"blog/internal/service"
	"context"
	"encoding/json"
	"log"

	"blog/pkg/kafka"
)

// 创建通知消息处理器
func NewNotificationHandler(artLikeService *service.ArticleLikeService) kafka.MessageHandler {
	return func(ctx context.Context, key string, value []byte) error {
		// 1. 解析消息
		var msg message.NotificationMsg
		if err := json.Unmarshal(value, &msg); err != nil {
			log.Printf("[MQ] 解析通知消息失败: %v", err)
			return err
		}
		return artLikeService.CreateLikeNotification(ctx, &msg)
	}
}
