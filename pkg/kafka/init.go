// Package kafka 封装 Kafka 生产者、消费者与死信队列能力。
package kafka

import (
	"blog/internal/platform/config"
	"sync"
)

// 全局 Kafka 客户端单例及其初始化状态
var (
	globalClient *Client   // 全局唯一的 Kafka 客户端实例
	once         sync.Once // 保证初始化逻辑只执行一次
	initErr      error     // 初始化过程中产生的错误
)

// 初始化全局 Kafka 客户端
func InitGlobalClient(cfg config.Kafka) error {
	// sync.Once 确保只初始化一次 Kafka 客户端
	once.Do(func() {
		globalClient, initErr = NewClient(cfg)
	})
	return initErr
}

// 获取全局 Kafka 客户端
// 如果未初始化，返回 nil
func GetGlobalClient() *Client {
	return globalClient
}

// 关闭全局 Kafka 客户端
// 如果未初始化，返回 nil
func CloseGlobalClient() error {
	if globalClient != nil {
		return globalClient.Close()
	}
	return nil
}
