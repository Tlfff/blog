// Package config 提供服务级配置与伙伴服务发现配置。
package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// Service 描述当前服务的身份、监听地址和治理参数。
type Service struct {
	Name       string        `yaml:"name"`
	ID         string        `yaml:"id"`
	Enabled    bool          `yaml:"enabled"`
	GRPCAddr   string        `yaml:"grpc_addr"`
	HealthAddr string        `yaml:"health_addr"`
	GRPCTimeout string       `yaml:"grpc_timeout"`
	RedisPrefix string       `yaml:"redis_prefix"`
	KafkaGroups KafkaGroups  `yaml:"kafka_groups"`
	Peers      []Peer        `yaml:"peers"`
}

// KafkaGroups 允许服务覆盖默认 consumer group，避免多服务争抢同一 group。
type KafkaGroups struct {
	Notification string `yaml:"notification"`
	ViewHistory  string `yaml:"view_history"`
}

// Peer 描述内部 gRPC 伙伴服务。
type Peer struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
}

// Load 从 YAML 文件加载服务级配置。
func Load(path string) (*Service, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取服务配置失败: %w", err)
	}
	var cfg Service
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析服务配置失败: %w", err)
	}
	if cfg.Name == "" || cfg.ID == "" {
		return nil, fmt.Errorf("服务配置缺少 name 或 id")
	}
	return &cfg, nil
}

// PeerAddr 按名称查找伙伴服务地址。
func (s *Service) PeerAddr(name string) string {
	for _, peer := range s.Peers {
		if peer.Name == name {
			return peer.Addr
		}
	}
	return ""
}
