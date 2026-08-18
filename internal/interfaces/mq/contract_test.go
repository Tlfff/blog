package mq

import "testing"

func TestKafkaHandlerRegistrationContract(t *testing.T) {
	handlers := RegisterHandlers(nil, nil)
	for _, key := range []string{"notification", "view_history"} {
		if _, ok := handlers[key]; !ok {
			t.Errorf("缺少 Kafka 处理器: %s", key)
		}
	}
}
