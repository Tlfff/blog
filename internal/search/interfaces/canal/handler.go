package canal

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	protocol "github.com/withlin/canal-go/protocol"
	entry "github.com/withlin/canal-go/protocol/entry"
	"google.golang.org/protobuf/proto"
)

const mysqlDateTimeLayout = "2006-01-02 15:04:05" // MySQL DATETIME 文本格式

var mysqlLocation = time.FixedZone("Asia/Shanghai", 8*60*60) // MySQL DATETIME 使用的固定东八区时区

// ChangeUsecase 定义文章搜索变更批量处理能力。
type ChangeUsecase interface {
	// HandleChanges 按原始顺序处理文章变更。
	HandleChanges(ctx context.Context, changes []searchdomain.ArticleChange) error
}

// Handler 把 Canal 原始批次转换为 Search 文章变更。
type Handler struct {
	usecase ChangeUsecase // 文章搜索增量同步用例
	schema  string        // 目标 MySQL 数据库名
	table   string        // 目标文章表名称
}

// NewHandler 创建 Search Canal 批次处理器。
func NewHandler(usecase ChangeUsecase, schema, table string) *Handler {
	// 1. 保存业务用例和目标表范围
	return &Handler{usecase: usecase, schema: strings.TrimSpace(schema), table: strings.TrimSpace(table)}
}

// HandleBatch 解析 Canal 批次并按原始顺序处理文章变更。
func (h *Handler) HandleBatch(ctx context.Context, message *protocol.Message) error {
	// 1. 空批次无需调用业务用例
	if message == nil || len(message.Entries) == 0 {
		return nil
	}
	if h == nil || h.usecase == nil {
		return searchdomain.ErrSearchUnavailable
	}

	// 2. 过滤事务和非目标表事件并解析 ROWDATA
	changes := make([]searchdomain.ArticleChange, 0)
	for index := range message.Entries {
		canalEntry := &message.Entries[index]
		// 2.1 过滤事务开始和结束等控制事件
		if canalEntry.GetEntryType() == entry.EntryType_TRANSACTIONBEGIN || canalEntry.GetEntryType() == entry.EntryType_TRANSACTIONEND {
			continue
		}
		header := canalEntry.GetHeader()
		// 2.2 过滤非目标表事件
		if header == nil || header.GetSchemaName() != h.schema || header.GetTableName() != h.table {
			continue
		}

		// 2.3 解析变更的行数据并跳过 DDL 结构变更
		rowChange := new(entry.RowChange)
		if err := proto.Unmarshal(canalEntry.GetStoreValue(), rowChange); err != nil {
			return fmt.Errorf("解析 Canal 文章行变更失败: %w", err)
		}
		if rowChange.GetIsDdl() {
			continue
		}

		// 3. 按行顺序转换 INSERT、UPDATE 和 DELETE
		changeType, err := mapChangeType(rowChange.GetEventType())
		if err != nil {
			return err
		}
		for _, row := range rowChange.GetRowDatas() {
			// 把每一行的修改前数据、修改后数据和变化字段转换成 ArticleChange
			change, err := mapArticleChange(changeType, row)
			// 任意一行解析失败，整批消息处理失败，之后由 Canal Client rollback
			if err != nil {
				return err
			}
			changes = append(changes, change)
		}
	}

	// 4. 目标批次没有文章行时直接成功，否则交给应用用例处理文章变更
	if len(changes) == 0 {
		return nil
	}

	return h.usecase.HandleChanges(ctx, changes)
}

// mapChangeType 把 Canal 事件类型转换为 Search 变更类型。
func mapChangeType(eventType entry.EventType) (searchdomain.ChangeType, error) {
	// 1. 只接受文章行级新增、更新和删除事件
	switch eventType {
	case entry.EventType_INSERT:
		return searchdomain.ChangeTypeInsert, nil
	case entry.EventType_UPDATE:
		return searchdomain.ChangeTypeUpdate, nil
	case entry.EventType_DELETE:
		return searchdomain.ChangeTypeDelete, nil
	default:
		return 0, fmt.Errorf("%w: %s", searchdomain.ErrChangeTypeInvalid, eventType.String())
	}
}

// mapArticleChange 把 Canal 单行数据转换为 Search 文章变更。
func mapArticleChange(changeType searchdomain.ChangeType, row *entry.RowData) (searchdomain.ArticleChange, error) {
	// 1. 校验行数据并解析变更前后文章
	if row == nil {
		return searchdomain.ArticleChange{}, fmt.Errorf("Canal 文章行数据不能为空")
	}
	before, err := mapSourceArticle(row.GetBeforeColumns())
	if err != nil {
		return searchdomain.ArticleChange{}, err
	}
	after, err := mapSourceArticle(row.GetAfterColumns())
	if err != nil {
		return searchdomain.ArticleChange{}, err
	}

	// 2. 从 UPDATE 后置字段标记收集实际变化字段
	changedFields := make(map[string]bool)
	for _, column := range row.GetAfterColumns() {
		if column.GetUpdated() {
			changedFields[column.GetName()] = true
		}
	}
	return searchdomain.ArticleChange{
		Type: changeType, Before: before, After: after, ChangedFields: changedFields,
	}, nil
}

// mapSourceArticle 把 Canal 列集合转换为搜索来源文章。
func mapSourceArticle(columns []*entry.Column) (searchdomain.SourceArticle, error) {
	// 1. 空列集合用于 INSERT 前值或 DELETE 后值
	if len(columns) == 0 {
		return searchdomain.SourceArticle{}, nil
	}
	values := make(map[string]string, len(columns))
	for _, column := range columns {
		values[column.GetName()] = column.GetValue()
	}

	// 2. 解析文章 ID、状态和更新时间
	articleID, err := strconv.ParseUint(values["id"], 10, 64)
	if err != nil {
		return searchdomain.SourceArticle{}, fmt.Errorf("解析 Canal 文章 ID %q 失败: %w", values["id"], err)
	}
	if articleID == 0 {
		return searchdomain.SourceArticle{}, fmt.Errorf("解析 Canal 文章 ID %q 失败: ID 必须大于 0", values["id"])
	}
	statusValue, err := strconv.ParseInt(values["status"], 10, 8)
	if err != nil {
		return searchdomain.SourceArticle{}, fmt.Errorf("解析 Canal 文章 %d 状态 %q 失败: %w", articleID, values["status"], err)
	}
	updatedTime, err := time.ParseInLocation(mysqlDateTimeLayout, values["updated_time"], mysqlLocation)
	if err != nil {
		return searchdomain.SourceArticle{}, fmt.Errorf("解析 Canal 文章 %d 更新时间 %q 失败: %w", articleID, values["updated_time"], err)
	}

	// 3. 组装搜索来源文章完整行数据
	return searchdomain.SourceArticle{
		ID: articleID, Title: values["title"], Content: values["content"], Tags: values["tags"],
		Status: searchdomain.ArticleSourceStatus(statusValue), UpdatedTime: updatedTime,
	}, nil
}
