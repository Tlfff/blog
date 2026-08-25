package kafka

import (
	"blog/internal/platform/config"
	"sync"
)

var (
	globalClient *Client
	once         sync.Once
	initErr      error
)

// InitGlobalClient 初始化全局 Kafka 客户端。
func InitGlobalClient(cfg config.Kafka) error {
	// sync.Once 确保只初始化一次 Kafka 客户端
	once.Do(func() {
		globalClient, initErr = NewClient(cfg)
	})
	return initErr
}

// GetGlobalClient 获取全局 Kafka 客户端。
// 如果未初始化，返回 nil
func GetGlobalClient() *Client {
	return globalClient
}

// CloseGlobalClient 关闭全局 Kafka 客户端。
// 如果未初始化，返回 nil
func CloseGlobalClient() error {
	if globalClient != nil {
		return globalClient.Close()
	}
	return nil
}
