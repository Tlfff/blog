package app

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"errors"
	"testing"
)

// fakeIndexWriter 记录增量同步执行的索引操作。
type fakeIndexWriter struct {
	upserts []searchdomain.ArticleDocument // 已写入的文章文档
	deletes []uint64                       // 已删除的文章 ID
	err     error                          // 预设索引错误
}

// CreateIndex 满足 IndexWriter 接口。
func (f *fakeIndexWriter) CreateIndex(context.Context, string) error { return f.err }

// BulkUpsert 记录批量写入文档。
func (f *fakeIndexWriter) BulkUpsert(_ context.Context, _ string, documents []searchdomain.ArticleDocument) error {
	// 1. 保存增量写入记录
	f.upserts = append(f.upserts, documents...)
	return f.err
}

// DeleteDocuments 记录批量删除 ID。
func (f *fakeIndexWriter) DeleteDocuments(_ context.Context, _ string, articleIDs []uint64) error {
	// 1. 保存增量删除记录
	f.deletes = append(f.deletes, articleIDs...)
	return f.err
}

// Refresh 满足 IndexWriter 接口。
func (f *fakeIndexWriter) Refresh(context.Context, string) error { return f.err }

// Count 满足 IndexWriter 接口。
func (f *fakeIndexWriter) Count(context.Context, string) (uint64, error) { return 0, f.err }

// SwitchAlias 满足 IndexWriter 接口。
func (f *fakeIndexWriter) SwitchAlias(context.Context, string, string) error { return f.err }

// DeleteIndex 满足 IndexWriter 接口。
func (f *fakeIndexWriter) DeleteIndex(context.Context, string) error { return f.err }

// TestSyncServiceHandleChange 验证文章状态和字段变化对应的索引操作。
func TestSyncServiceHandleChange(t *testing.T) {
	// 1. 定义需要 upsert、delete 和 ignore 的事件
	tests := []struct {
		name        string                     // 测试场景名称
		change      searchdomain.ArticleChange // 文章变更
		wantUpserts int                        // 预期写入次数
		wantDeletes int                        // 预期删除次数
	}{
		{name: "新增已发表文章", change: searchdomain.ArticleChange{Type: searchdomain.ChangeTypeInsert, After: publishedArticle(1)}, wantUpserts: 1},
		{name: "新增草稿", change: searchdomain.ArticleChange{Type: searchdomain.ChangeTypeInsert, After: draftArticle(2)}},
		{name: "草稿发布", change: searchdomain.ArticleChange{Type: searchdomain.ChangeTypeUpdate, Before: draftArticle(3), After: publishedArticle(3), ChangedFields: map[string]bool{"status": true}}, wantUpserts: 1},
		{name: "退出发表状态", change: searchdomain.ArticleChange{Type: searchdomain.ChangeTypeUpdate, Before: publishedArticle(4), After: draftArticle(4), ChangedFields: map[string]bool{"status": true}}, wantDeletes: 1},
		{name: "更新标题", change: searchdomain.ArticleChange{Type: searchdomain.ChangeTypeUpdate, Before: publishedArticle(5), After: publishedArticle(5), ChangedFields: map[string]bool{"title": true}}, wantUpserts: 1},
		{name: "只更新统计", change: searchdomain.ArticleChange{Type: searchdomain.ChangeTypeUpdate, Before: publishedArticle(6), After: publishedArticle(6), ChangedFields: map[string]bool{"view_count": true}}},
		{name: "物理删除", change: searchdomain.ArticleChange{Type: searchdomain.ChangeTypeDelete, Before: publishedArticle(7)}, wantDeletes: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := &fakeIndexWriter{}
			service := NewSyncService(writer, NewDocumentFactory(fakeExtractor{content: "正文"}))
			if err := service.HandleChange(context.Background(), test.change); err != nil {
				t.Fatalf("处理文章变更失败: %v", err)
			}
			if len(writer.upserts) != test.wantUpserts || len(writer.deletes) != test.wantDeletes {
				t.Fatalf("索引操作不符合预期: upserts=%v deletes=%v", writer.upserts, writer.deletes)
			}
		})
	}
}

// TestSyncServiceStopsOnFailure 验证批次按顺序处理并在失败时停止。
func TestSyncServiceStopsOnFailure(t *testing.T) {
	// 1. 准备持续失败的索引写入器
	writeErr := errors.New("写入失败")
	writer := &fakeIndexWriter{err: writeErr}
	service := NewSyncService(writer, NewDocumentFactory(fakeExtractor{content: "正文"}))

	// 2. 第一条失败后返回错误且不继续确认后续语义
	err := service.HandleChanges(context.Background(), []searchdomain.ArticleChange{
		{Type: searchdomain.ChangeTypeInsert, After: publishedArticle(1)},
		{Type: searchdomain.ChangeTypeInsert, After: publishedArticle(2)},
	})
	if !errors.Is(err, writeErr) || len(writer.upserts) != 1 {
		t.Fatalf("同步失败处理不符合预期: err=%v upserts=%d", err, len(writer.upserts))
	}
}

// TestSyncServiceRepeatedEventIsIdempotent 验证重复文章事件使用同一文档 ID 覆盖写入。
func TestSyncServiceRepeatedEventIsIdempotent(t *testing.T) {
	// 1. 准备同一篇已发表文章的重复事件
	writer := &fakeIndexWriter{}
	service := NewSyncService(writer, NewDocumentFactory(fakeExtractor{content: "正文"}))
	change := searchdomain.ArticleChange{Type: searchdomain.ChangeTypeInsert, After: publishedArticle(10)}

	// 2. 重复处理仍只产生相同文章 ID 的幂等覆盖语义
	if err := service.HandleChanges(context.Background(), []searchdomain.ArticleChange{change, change}); err != nil {
		t.Fatalf("处理重复文章事件失败: %v", err)
	}
	if len(writer.upserts) != 2 || writer.upserts[0].ArticleID != 10 || writer.upserts[1].ArticleID != 10 {
		t.Fatalf("重复事件文档 ID 不一致: %+v", writer.upserts)
	}
}

// publishedArticle 返回已发表来源文章。
func publishedArticle(id uint64) searchdomain.SourceArticle {
	// 1. 构造可进入公开搜索索引的文章
	return searchdomain.SourceArticle{ID: id, Title: "标题", Content: "正文", Tags: "Go", Status: searchdomain.ArticleStatusPublished}
}

// draftArticle 返回草稿来源文章。
func draftArticle(id uint64) searchdomain.SourceArticle {
	// 1. 构造不进入公开搜索索引的草稿
	return searchdomain.SourceArticle{ID: id, Status: searchdomain.ArticleStatusDraft}
}
