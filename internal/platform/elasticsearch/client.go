// Package elasticsearch 承载平台层 Elasticsearch 客户端初始化与资源管理能力。
package elasticsearch

import (
	platformconfig "blog/internal/platform/config"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	elastic "github.com/elastic/go-elasticsearch/v8"
)

// Client 表示带有可关闭 HTTP 连接池的 Elasticsearch 客户端。
type Client struct {
	API       *elastic.Client // Elasticsearch 官方低层客户端
	transport *http.Transport // 客户端持有的 HTTP 连接池
}

// NewClient 根据平台配置创建 Elasticsearch 客户端。
func NewClient(cfg platformconfig.Elasticsearch) (*Client, error) {
	// 1. 校验连接配置并创建独立 HTTP 连接池
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.GetRequestTimeoutMS()) * time.Millisecond,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}

	// 2. 创建官方低层客户端，不主动探测集群以隔离 HTTP Server 启动
	api, err := elastic.NewClient(elastic.Config{
		Addresses:     []string{strings.TrimSpace(cfg.Addr)},
		Username:      cfg.Username,
		Password:      cfg.Password,
		Transport:     transport,
		MaxRetries:    3,
		RetryOnStatus: []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout},
	})
	if err != nil {
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("创建 Elasticsearch 客户端失败: %w", err)
	}
	return &Client{API: api, transport: transport}, nil
}

// Close 关闭 Elasticsearch 客户端持有的空闲 HTTP 连接。
func (c *Client) Close() error {
	// 1. 客户端为空时保持幂等成功
	if c == nil || c.transport == nil {
		return nil
	}
	// 2. 关闭空闲连接，正在处理的请求由其上下文负责终止
	c.transport.CloseIdleConnections()
	return nil
}
