package mq

import (
	"blog/internal/service"
	"blog/pkg/kafka"
)

// 注册所有 Kafka 消息处理器
func RegisterHandlers(
	artLikeService *service.ArticleLikeService,
	historyService *service.ArticleViewHistoryService,
) map[string]kafka.MessageHandler {
	return map[string]kafka.MessageHandler{
		"notification": NewNotificationHandler(artLikeService),
		"view_history": NewViewHistoryHandler(historyService),
	}
}
