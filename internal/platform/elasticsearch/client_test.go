package elasticsearch

import (
	platformconfig "blog/internal/platform/config"
	"testing"
)

// TestNewClient 验证 Elasticsearch 客户端配置校验和幂等关闭。
func TestNewClient(t *testing.T) {
	// 1. 缺少地址时拒绝创建客户端
	if _, err := NewClient(platformconfig.Elasticsearch{}); err == nil {
		t.Fatal("缺少 Elasticsearch 地址时未返回错误")
	}

	// 2. 合法地址不依赖在线集群即可完成客户端构造
	client, err := NewClient(platformconfig.Elasticsearch{Addr: "http://127.0.0.1:9200"})
	if err != nil {
		t.Fatalf("创建 Elasticsearch 客户端失败: %v", err)
	}
	if client.API == nil {
		t.Fatal("Elasticsearch 官方客户端为空")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("关闭 Elasticsearch 客户端失败: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("重复关闭 Elasticsearch 客户端失败: %v", err)
	}
}
