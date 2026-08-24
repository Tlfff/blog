package mq

import (
	"context"
	"testing"
)

// TestKafkaHandlerRegistrationContract 验证两个基线 Topic 均已注册。
func TestKafkaHandlerRegistrationContract(t *testing.T) {
	handler := func(context.Context, string, []byte) error { return nil }
	handlers := RegisterHandlers(handler, handler)
	for _, key := range []string{"notification", "view_history"} {
		if _, ok := handlers[key]; !ok {
			t.Errorf("缺少 Kafka 处理器: %s", key)
		}
	}
}
