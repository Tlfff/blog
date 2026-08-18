package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestKafkaContractConfig(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件")
	}
	configPath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "config", "config.yaml")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	notification := cfg.Kafka.Topics["notification"]
	if notification.Name != "notification" || notification.GroupID != "blog_notification_consumer" {
		t.Fatalf("notification topic 契约被改变: %+v", notification)
	}

	viewHistory := cfg.Kafka.Topics["view_history"]
	if viewHistory.Name != "view_history" || viewHistory.GroupID != "blog_view_consumer" {
		t.Fatalf("view_history topic 契约被改变: %+v", viewHistory)
	}

	if cfg.Kafka.GetDeadLetterTopic() != "dead_letter" {
		t.Fatalf("dead_letter topic 契约被改变: %s", cfg.Kafka.GetDeadLetterTopic())
	}
}
