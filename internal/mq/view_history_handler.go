package mq

import (
	"blog/internal/message"
	"blog/internal/service"
	"context"
	"encoding/json"
	"log"

	"blog/pkg/kafka"
)

// 创建浏览历史消息处理器
func NewViewHistoryHandler(historyService *service.ArticleViewHistoryService) kafka.MessageHandler {
	return func(ctx context.Context, key string, value []byte) error {
		// 1. 解析消息
		var msg message.ViewHistoryMsg
		if err := json.Unmarshal(value, &msg); err != nil {
			log.Printf("[MQ] 解析浏览历史消息失败: %v", err)
			return err
		}

		// 2. 调用 service 处理浏览历史
		err := historyService.CreateViewHistory(ctx, msg.UserID, msg.ArticleID, msg.CreatedTime)
		if err != nil {
			log.Printf("[MQ] 处理浏览历史失败: %v", err)
			return err
		}

		log.Printf("[MQ] 浏览历史处理成功, user: %d, article: %d", msg.UserID, msg.ArticleID)
		return nil
	}
}
