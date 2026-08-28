// Package canal 承载平台层 Canal Client 的连接、批次确认和重试能力。
package canal

import (
	platformconfig "blog/internal/platform/config"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	canalclient "github.com/withlin/canal-go/client"
	protocol "github.com/withlin/canal-go/protocol"
)

var errClientClosed = errors.New("Canal Client 已关闭")

// BatchHandler 定义 Canal 批次的业务处理能力。
type BatchHandler interface {
	// HandleBatch 处理一个尚未确认的 Canal 批次。
	HandleBatch(ctx context.Context, message *protocol.Message) error
}

// connector 描述底层 Canal Connector 所需的最小协议能力。
type connector interface {
	// Connect 建立 Canal 连接。
	Connect() error
	// DisConnection 关闭 Canal 连接。
	DisConnection() error
	// Subscribe 订阅 binlog 过滤表达式。
	Subscribe(filter string) error
	// GetWithOutAck 拉取尚未自动确认的批次。
	GetWithOutAck(batchSize int32, timeout *int64, units *int32) (*protocol.Message, error)
	// Ack 确认批次已经处理成功。
	Ack(batchID int64) error
	// RollBack 回滚尚未成功处理的批次。
	RollBack(batchID int64) error
}

// connectorFactory 创建可重新连接的底层 Canal Connector。
type connectorFactory func() connector

// Client 表示可长期运行的 Canal 批次消费者。
type Client struct {
	config    platformconfig.Canal // Canal 运行配置
	handler   BatchHandler         // Canal 批次业务处理器
	newClient connectorFactory     // 底层 Connector 工厂
	mutex     sync.Mutex           // 保护当前连接和关闭状态
	current   connector            // 当前正在使用的 Canal 连接
	closed    bool                 // 是否已显式关闭客户端
}

// NewClient 创建平台层 Canal Client。
func NewClient(cfg platformconfig.Canal, handler BatchHandler) (*Client, error) {
	// 1. 校验连接配置与业务处理器
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("Canal 批次处理器不能为空")
	}

	// 2. 构建可在断线后重新创建的底层 Connector
	return newClient(cfg, handler, func() connector {
		client := canalclient.NewSimpleCanalConnector(
			cfg.Host,
			cfg.Port,
			cfg.Username,
			cfg.Password,
			cfg.Destination,
			cfg.GetSocketTimeoutMS(),
			cfg.GetIdleTimeoutMS(),
		)
		return client
	}), nil
}

// newClient 使用指定 Connector 工厂创建 Canal Client，供测试替换网络实现。
func newClient(cfg platformconfig.Canal, handler BatchHandler, factory connectorFactory) *Client {
	// 1. 保存运行依赖并延迟建立网络连接
	return &Client{config: cfg, handler: handler, newClient: factory}
}

// Run 持续拉取、处理并确认 Canal 批次，直到上下文取消或客户端关闭。
func (c *Client) Run(ctx context.Context) error {
	// 1. 校验运行状态
	if c == nil {
		return errors.New("Canal Client 不能为空")
	}
	if ctx == nil {
		return errors.New("Canal 运行上下文不能为空")
	}

	// 2. 建立连接并在断线或处理失败后按上限退避重试
	reconnectWait := time.Duration(c.config.GetReconnectMinWaitMS()) * time.Millisecond
	processWait := reconnectWait
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		conn, err := c.connect()
		if err != nil {
			if errors.Is(err, errClientClosed) {
				return nil
			}
			log.Printf("[WARN] 连接 Canal Server 失败，等待重试: %v", err)
			if waitErr := waitContext(ctx, reconnectWait); waitErr != nil {
				return nil
			}
			reconnectWait = nextWait(reconnectWait, time.Duration(c.config.GetReconnectMaxWaitMS())*time.Millisecond)
			continue
		}
		reconnectWait = time.Duration(c.config.GetReconnectMinWaitMS()) * time.Millisecond

		// 3. 拉取未确认批次，空批次等待后继续
		message, err := conn.GetWithOutAck(c.config.GetBatchSize(), nil, nil)
		if err != nil {
			log.Printf("[WARN] 拉取 Canal 批次失败，重新建立连接: %v", err)
			c.disconnect(conn)
			continue
		}
		if message == nil || message.Id == -1 || len(message.Entries) == 0 {
			processWait = time.Duration(c.config.GetReconnectMinWaitMS()) * time.Millisecond
			if err := waitContext(ctx, time.Duration(c.config.GetEmptyWaitMS())*time.Millisecond); err != nil {
				return nil
			}
			continue
		}

		// 4. 业务处理成功后确认批次，失败时回滚并重试同一位点
		if err := c.handler.HandleBatch(ctx, message); err != nil {
			rollbackErr := conn.RollBack(message.Id)
			log.Printf("[ERROR] 处理 Canal 批次 %d 失败，准备回滚重试: %v", message.Id, err)
			if rollbackErr != nil {
				log.Printf("[ERROR] 回滚 Canal 批次 %d 失败，重新建立连接: %v", message.Id, rollbackErr)
				c.disconnect(conn) // 断开异常连接，下一轮循环重新连接 Canal Server
			}
			if waitErr := waitContext(ctx, processWait); waitErr != nil {
				return nil // 等待期间收到进程退出信号，结束消费循环
			}
			// 增加下一次失败后的等待时间，但不超过配置的最大值
			processWait = nextWait(processWait, time.Duration(c.config.GetProcessRetryMaxWaitMS())*time.Millisecond)
			continue
		}
		// 业务已经成功，但 Ack 失败，不能确定 Canal Server 是否记录了确认结果
		if err := conn.Ack(message.Id); err != nil {
			log.Printf("[ERROR] 确认 Canal 批次 %d 失败，重新建立连接: %v", message.Id, err)
			c.disconnect(conn) // 断开异常连接，下一轮循环重新连接 Canal Server
			continue
		}
		// 批次成功处理并 Ack 后，重置业务重试等待时间
		processWait = time.Duration(c.config.GetReconnectMinWaitMS()) * time.Millisecond
	}
}

// Close 关闭当前 Canal 连接并阻止建立新连接。
func (c *Client) Close() error {
	// 1. 原子标记关闭并取出当前连接
	if c == nil {
		return nil
	}
	c.mutex.Lock()
	if c.closed {
		c.mutex.Unlock()
		return nil
	}
	c.closed = true
	conn := c.current
	c.current = nil
	c.mutex.Unlock()

	// 2. 关闭底层连接以解除阻塞中的拉取请求
	if conn != nil {
		if err := conn.DisConnection(); err != nil {
			return fmt.Errorf("关闭 Canal 连接失败: %w", err)
		}
	}
	return nil
}

// connect 获取或建立当前 Canal 连接。
func (c *Client) connect() (connector, error) {
	// 1. 已有连接时直接复用
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.closed {
		return nil, errClientClosed
	}
	if c.current != nil {
		return c.current, nil
	}

	// 2. 创建连接并订阅目标表
	conn := c.newClient()
	if conn == nil {
		return nil, errors.New("Canal Connector 工厂返回空连接")
	}
	if err := conn.Connect(); err != nil {
		_ = conn.DisConnection()
		return nil, fmt.Errorf("连接 Canal Server 失败: %w", err)
	}
	if c.config.Filter != "" {
		if err := conn.Subscribe(c.config.Filter); err != nil {
			_ = conn.DisConnection()
			return nil, fmt.Errorf("订阅 Canal binlog 失败: %w", err)
		}
	}
	c.current = conn
	return conn, nil
}

// disconnect 移除并关闭指定的失效连接。
func (c *Client) disconnect(conn connector) {
	// 1. 仅清除仍是当前连接的实例
	c.mutex.Lock()
	if c.current == conn {
		c.current = nil
	}
	c.mutex.Unlock()

	// 2. 尽力关闭失效连接，后续循环会重新建立连接
	if conn != nil {
		_ = conn.DisConnection()
	}
}

// waitContext 等待指定时长，并允许上下文提前取消。
func waitContext(ctx context.Context, duration time.Duration) error {
	// 1. 非正等待时间直接检查上下文
	if duration <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()

	// 2. 在等待完成或上下文取消时返回
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// nextWait 计算不超过上限的指数退避时间。
func nextWait(current, maximum time.Duration) time.Duration {
	// 1. 防止未配置或非法等待时间造成忙循环
	if current <= 0 {
		current = time.Millisecond
	}
	if maximum <= 0 || current >= maximum {
		return maximum
	}

	// 2. 加倍等待时间并限制在最大值内
	next := current * 2
	if next > maximum {
		return maximum
	}
	return next
}
