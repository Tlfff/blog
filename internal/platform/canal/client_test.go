package canal

import (
	platformconfig "blog/internal/platform/config"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	protocol "github.com/withlin/canal-go/protocol"
	entry "github.com/withlin/canal-go/protocol/entry"
)

// fakeConnector 记录 Canal Client 对底层协议的调用。
type fakeConnector struct {
	mutex           sync.Mutex          // 保护测试调用记录
	connectErr      error               // 建立连接时返回的错误
	getErr          error               // 拉取批次时返回的错误
	messages        []*protocol.Message // 按顺序返回的 Canal 批次
	connectCount    int                 // Connect 调用次数
	disconnectCount int                 // DisConnection 调用次数
	subscribeCount  int                 // Subscribe 调用次数
	ackIDs          []int64             // 已确认的批次 ID
	rollbackIDs     []int64             // 已回滚的批次 ID
}

// Connect 记录连接调用并返回预设错误。
func (f *fakeConnector) Connect() error {
	// 1. 记录连接调用
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.connectCount++
	return f.connectErr
}

// DisConnection 记录关闭连接调用。
func (f *fakeConnector) DisConnection() error {
	// 1. 记录关闭调用
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.disconnectCount++
	return nil
}

// Subscribe 记录订阅调用。
func (f *fakeConnector) Subscribe(string) error {
	// 1. 记录订阅调用
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.subscribeCount++
	return nil
}

// GetWithOutAck 返回预设批次或空批次。
func (f *fakeConnector) GetWithOutAck(int32, *int64, *int32) (*protocol.Message, error) {
	// 1. 返回预设拉取错误
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	if len(f.messages) == 0 {
		return &protocol.Message{Id: -1}, nil
	}
	message := f.messages[0]
	if len(f.messages) > 1 {
		f.messages = f.messages[1:]
	}
	return message, nil
}

// Ack 记录已确认批次。
func (f *fakeConnector) Ack(batchID int64) error {
	// 1. 保存确认记录
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.ackIDs = append(f.ackIDs, batchID)
	return nil
}

// RollBack 记录已回滚批次。
func (f *fakeConnector) RollBack(batchID int64) error {
	// 1. 保存回滚记录
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.rollbackIDs = append(f.rollbackIDs, batchID)
	return nil
}

// handlerFunc 把函数适配为 BatchHandler。
type handlerFunc func(context.Context, *protocol.Message) error

// HandleBatch 调用测试处理函数。
func (f handlerFunc) HandleBatch(ctx context.Context, message *protocol.Message) error {
	// 1. 转发批次处理
	return f(ctx, message)
}

// TestClientAckAfterSuccess 验证成功处理后才确认批次。
func TestClientAckAfterSuccess(t *testing.T) {
	// 1. 准备单条非空批次和可取消上下文
	ctx, cancel := context.WithCancel(context.Background())
	fake := &fakeConnector{messages: []*protocol.Message{{Id: 7, Entries: []entry.Entry{{}}}}}
	client := newClient(testConfig(), handlerFunc(func(context.Context, *protocol.Message) error {
		cancel()
		return nil
	}), func() connector { return fake })

	// 2. 运行客户端并核对确认顺序
	if err := client.Run(ctx); err != nil {
		t.Fatalf("运行 Canal Client 失败: %v", err)
	}
	if len(fake.ackIDs) != 1 || fake.ackIDs[0] != 7 {
		t.Fatalf("成功批次未正确确认: %+v", fake.ackIDs)
	}
	if len(fake.rollbackIDs) != 0 {
		t.Fatalf("成功批次被意外回滚: %+v", fake.rollbackIDs)
	}
	if fake.subscribeCount != 1 {
		t.Fatalf("Canal 连接订阅次数不符合预期: %d", fake.subscribeCount)
	}
}

// TestClientRollbackAndRetry 验证处理失败后回滚并重试同一批次。
func TestClientRollbackAndRetry(t *testing.T) {
	// 1. 准备重复返回的同一批次和先失败后成功的处理器
	ctx, cancel := context.WithCancel(context.Background())
	message := &protocol.Message{Id: 8, Entries: []entry.Entry{{}}}
	fake := &fakeConnector{messages: []*protocol.Message{message}}
	handleCount := 0
	client := newClient(testConfig(), handlerFunc(func(context.Context, *protocol.Message) error {
		handleCount++
		if handleCount == 1 {
			return errors.New("索引写入失败")
		}
		cancel()
		return nil
	}), func() connector { return fake })

	// 2. 运行客户端并核对回滚后重新处理和确认
	if err := client.Run(ctx); err != nil {
		t.Fatalf("运行 Canal Client 失败: %v", err)
	}
	if len(fake.rollbackIDs) != 1 || fake.rollbackIDs[0] != 8 {
		t.Fatalf("失败批次未正确回滚: %+v", fake.rollbackIDs)
	}
	if handleCount != 2 || len(fake.ackIDs) != 1 || fake.ackIDs[0] != 8 {
		t.Fatalf("回滚批次未成功重试: handle=%d ack=%+v", handleCount, fake.ackIDs)
	}
}

// TestClientReconnect 验证连接失败后按工厂创建新连接重试。
func TestClientReconnect(t *testing.T) {
	// 1. 准备首次连接失败和第二次成功的 Connector
	ctx, cancel := context.WithCancel(context.Background())
	failed := &fakeConnector{connectErr: errors.New("网络不可用")}
	succeeded := &fakeConnector{messages: []*protocol.Message{{Id: 9, Entries: []entry.Entry{{}}}}}
	connectors := []connector{failed, succeeded}
	factoryCalls := 0
	client := newClient(testConfig(), handlerFunc(func(context.Context, *protocol.Message) error {
		cancel()
		return nil
	}), func() connector {
		connector := connectors[factoryCalls]
		factoryCalls++
		return connector
	})

	// 2. 运行客户端并确认已重连到第二个实例
	if err := client.Run(ctx); err != nil {
		t.Fatalf("重连 Canal Client 失败: %v", err)
	}
	if factoryCalls != 2 || len(succeeded.ackIDs) != 1 {
		t.Fatalf("Canal Client 未按预期重连: factory=%d ack=%+v", factoryCalls, succeeded.ackIDs)
	}
}

// TestClientCancelOnEmptyBatch 验证空批次等待能够响应上下文取消。
func TestClientCancelOnEmptyBatch(t *testing.T) {
	// 1. 准备持续返回空批次的客户端
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fake := &fakeConnector{}
	client := newClient(testConfig(), handlerFunc(func(context.Context, *protocol.Message) error {
		t.Fatal("空批次不应调用业务处理器")
		return nil
	}), func() connector { return fake })

	// 2. 已取消上下文应立即结束
	started := time.Now()
	if err := client.Run(ctx); err != nil {
		t.Fatalf("取消 Canal Client 失败: %v", err)
	}
	if time.Since(started) > 100*time.Millisecond {
		t.Fatalf("Canal Client 未及时响应取消: %v", time.Since(started))
	}
}

// testConfig 返回适合单元测试的最短等待 Canal 配置。
func testConfig() platformconfig.Canal {
	// 1. 使用 1 毫秒退避缩短测试时间
	return platformconfig.Canal{
		Host:                  "127.0.0.1",
		Port:                  11111,
		Destination:           "article_search",
		Filter:                "blog\\.articles",
		EmptyWaitMS:           1,
		ReconnectMinWaitMS:    1,
		ReconnectMaxWaitMS:    2,
		ProcessRetryMaxWaitMS: 2,
	}
}
