package canal

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"errors"
	"strconv"
	"testing"

	protocol "github.com/withlin/canal-go/protocol"
	entry "github.com/withlin/canal-go/protocol/entry"
	"google.golang.org/protobuf/proto"
)

// fakeChangeUsecase 记录 Canal Handler 转换后的文章变更。
type fakeChangeUsecase struct {
	changes []searchdomain.ArticleChange // 收到的文章变更
	err     error                        // 预设处理错误
}

// HandleChanges 记录变更并返回预设错误。
func (f *fakeChangeUsecase) HandleChanges(_ context.Context, changes []searchdomain.ArticleChange) error {
	// 1. 保存业务变更
	f.changes = append(f.changes, changes...)
	return f.err
}

// TestHandlerHandleBatch 验证 INSERT、UPDATE、DELETE 和非目标表过滤。
func TestHandlerHandleBatch(t *testing.T) {
	// 1. 准备目标表三类事件和一条非目标表事件
	usecase := &fakeChangeUsecase{}
	handler := NewHandler(usecase, "blog", "articles")
	message := &protocol.Message{Id: 1, Entries: make([]entry.Entry, 4)}
	setCanalEntry(t, &message.Entries[0], "blog", "articles", entry.EventType_INSERT, nil, articleColumns(1, 3, map[string]bool{}))
	setCanalEntry(t, &message.Entries[1], "blog", "articles", entry.EventType_UPDATE, articleColumns(2, 2, nil), articleColumns(2, 3, map[string]bool{"status": true}))
	setCanalEntry(t, &message.Entries[2], "blog", "articles", entry.EventType_DELETE, articleColumns(3, 3, nil), nil)
	setCanalEntry(t, &message.Entries[3], "blog", "users", entry.EventType_UPDATE, articleColumns(4, 3, nil), articleColumns(4, 3, nil))

	// 2. 处理批次并核对顺序、类型、状态和变化字段
	if err := handler.HandleBatch(context.Background(), message); err != nil {
		t.Fatalf("处理 Canal 批次失败: %v", err)
	}
	if len(usecase.changes) != 3 {
		t.Fatalf("目标文章变更数量不符合预期: %+v", usecase.changes)
	}
	if usecase.changes[0].Type != searchdomain.ChangeTypeInsert || usecase.changes[0].After.ID != 1 {
		t.Fatalf("INSERT 转换不符合预期: %+v", usecase.changes[0])
	}
	if usecase.changes[1].Type != searchdomain.ChangeTypeUpdate || !usecase.changes[1].ChangedFields["status"] || !usecase.changes[1].After.IsPublished() {
		t.Fatalf("UPDATE 转换不符合预期: %+v", usecase.changes[1])
	}
	if usecase.changes[2].Type != searchdomain.ChangeTypeDelete || usecase.changes[2].Before.ID != 3 {
		t.Fatalf("DELETE 转换不符合预期: %+v", usecase.changes[2])
	}
}

// TestHandlerPreservesUsecaseError 验证业务处理错误原样返回以触发 Canal rollback。
func TestHandlerPreservesUsecaseError(t *testing.T) {
	// 1. 准备可识别的应用错误
	usecaseErr := errors.New("ES 写入失败")
	handler := NewHandler(&fakeChangeUsecase{err: usecaseErr}, "blog", "articles")
	message := &protocol.Message{Entries: make([]entry.Entry, 1)}
	setCanalEntry(t, &message.Entries[0], "blog", "articles", entry.EventType_INSERT, nil, articleColumns(1, 3, nil))

	// 2. Handler 返回同一错误链
	if err := handler.HandleBatch(context.Background(), message); !errors.Is(err, usecaseErr) {
		t.Fatalf("业务错误链丢失: %v", err)
	}
}

// TestHandlerRejectsMalformedRows 验证非法 protobuf 和字段数据会阻止批次确认。
func TestHandlerRejectsMalformedRows(t *testing.T) {
	// 1. 定义损坏 protobuf 和非法文章 ID 场景
	malformedMessage := &protocol.Message{Entries: make([]entry.Entry, 1)}
	malformedMessage.Entries[0] = entry.Entry{
		Header:           &entry.Header{SchemaName: "blog", TableName: "articles"},
		EntryTypePresent: &entry.Entry_EntryType{EntryType: entry.EntryType_ROWDATA},
		StoreValue:       []byte("invalid"),
	}
	invalidIDMessage := &protocol.Message{Entries: make([]entry.Entry, 1)}
	setCanalEntry(t, &invalidIDMessage.Entries[0], "blog", "articles", entry.EventType_INSERT, nil, []*entry.Column{
		column("id", "invalid", false), column("status", "3", false), column("updated_time", "2026-08-27 12:00:00", false),
	})
	tests := []*protocol.Message{malformedMessage, invalidIDMessage}

	// 2. 任一畸形事件都返回错误且不调用应用用例
	for _, message := range tests {
		usecase := &fakeChangeUsecase{}
		handler := NewHandler(usecase, "blog", "articles")
		if err := handler.HandleBatch(context.Background(), message); err == nil {
			t.Fatal("畸形 Canal 事件未返回错误")
		}
		if len(usecase.changes) != 0 {
			t.Fatalf("畸形事件仍调用了应用用例: %+v", usecase.changes)
		}
	}
}

// setCanalEntry 填充包含单行数据的 Canal ROWDATA 事件。
//
// 参数说明：
//   - t：当前测试实例。
//   - target：待填充的 Canal Entry，不能为空。
//   - schema：事件所属 MySQL schema。
//   - table：事件所属数据表。
//   - eventType：Canal 行事件类型。
//   - before：变更前列集合。
//   - after：变更后列集合。
func setCanalEntry(
	t *testing.T,
	target *entry.Entry,
	schema string,
	table string,
	eventType entry.EventType,
	before []*entry.Column,
	after []*entry.Column,
) {
	// 1. 序列化 RowChange 并组装 ROWDATA Entry
	t.Helper()
	rowChange := &entry.RowChange{
		EventTypePresent: &entry.RowChange_EventType{EventType: eventType},
		RowDatas:         []*entry.RowData{{BeforeColumns: before, AfterColumns: after}},
	}
	storeValue, err := proto.Marshal(rowChange)
	if err != nil {
		t.Fatalf("序列化 Canal RowChange 失败: %v", err)
	}
	*target = entry.Entry{
		Header:           &entry.Header{SchemaName: schema, TableName: table},
		EntryTypePresent: &entry.Entry_EntryType{EntryType: entry.EntryType_ROWDATA},
		StoreValue:       storeValue,
	}
}

// articleColumns 创建一篇文章的 Canal 完整列集合。
func articleColumns(id uint64, status int8, updated map[string]bool) []*entry.Column {
	// 1. 按文章表字段创建完整行，并设置变化标记
	return []*entry.Column{
		column("id", uintString(id), updated["id"]),
		column("title", "文章标题", updated["title"]),
		column("content", "正文", updated["content"]),
		column("tags", "Go,ES", updated["tags"]),
		column("status", intString(int64(status)), updated["status"]),
		column("updated_time", "2026-08-27 12:00:00", updated["updated_time"]),
		column("view_count", "1", updated["view_count"]),
	}
}

// column 创建 Canal 列。
func column(name, value string, updated bool) *entry.Column {
	// 1. 组装列名、值和变化标记
	return &entry.Column{Name: name, Value: value, Updated: updated}
}

// uintString 把无符号整数转换为十进制文本。
func uintString(value uint64) string {
	// 1. 返回 Canal 列使用的十进制字符串
	return strconv.FormatUint(value, 10)
}

// intString 把有符号整数转换为十进制文本。
func intString(value int64) string {
	// 1. 返回 Canal 列使用的十进制字符串
	return strconv.FormatInt(value, 10)
}
