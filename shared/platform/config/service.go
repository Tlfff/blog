// Package config 提供服务级配置与伙伴服务发现配置。
package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// Service 描述当前服务的身份、监听地址和治理参数。
type Service struct {
	Name        string      `yaml:"name"`         // 服务名称，如 blog-identity
	ID          string      `yaml:"id"`           // 服务唯一标识，用于内部gRPC身份校验
	Enabled     bool        `yaml:"enabled"`      // 是否启用该服务
	GRPCAddr    string      `yaml:"grpc_addr"`    // 内部gRPC监听地址
	HealthAddr  string      `yaml:"health_addr"`  // 健康检查监听地址
	GRPCTimeout string      `yaml:"grpc_timeout"` // 调用伙伴服务的超时时间，如 3s
	RedisPrefix string      `yaml:"redis_prefix"` // Redis key 前缀，用于隔离各服务数据
	KafkaGroups KafkaGroups `yaml:"kafka_groups"` // Kafka 消费组覆盖配置
	Peers       []Peer      `yaml:"peers"`        // 内部gRPC伙伴服务列表
}

// KafkaGroups 允许服务覆盖默认 consumer group，避免多服务争抢同一 group。
type KafkaGroups struct {
	Notification string `yaml:"notification"` // 通知事件消费组名
	ViewHistory  string `yaml:"view_history"` // 浏览历史事件消费组名
}

// Peer 描述内部 gRPC 伙伴服务。
type Peer struct {
	Name string `yaml:"name"` // 伙伴服务名称
	Addr string `yaml:"addr"` // 伙伴服务gRPC地址
}

// 从 YAML 文件加载服务级配置
func Load(path string) (*Service, error) {
	// 1. 读取配置文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取服务配置失败: %w", err)
	}
	// 2. 反序列化 YAML 到配置结构
	var cfg Service
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析服务配置失败: %w", err)
	}
	// 3. 校验必填项：服务名与服务ID
	if cfg.Name == "" || cfg.ID == "" {
		return nil, fmt.Errorf("服务配置缺少 name 或 id")
	}
	return &cfg, nil
}

// 按服务名称查找伙伴服务的gRPC地址，未配置时返回空字符串
func (s *Service) PeerAddr(name string) string {
	// 1. 遍历伙伴列表按名称匹配
	for _, peer := range s.Peers {
		if peer.Name == name {
			return peer.Addr
		}
	}
	// 2. 未匹配到则返回空字符串，由调用方决定降级策略
	return ""
}
