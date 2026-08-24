// Package mq 负责统一注册各上下文暴露的 Kafka Handler。
package mq

import "blog/internal/platform/kafka"

// RegisterHandlers 注册当前基线已接线的通知和浏览历史 Handler。
func RegisterHandlers(notificationHandler, viewHistoryHandler kafka.MessageHandler) map[string]kafka.MessageHandler {
	return map[string]kafka.MessageHandler{
		"notification": notificationHandler,
		"view_history": viewHistoryHandler,
	}
}
