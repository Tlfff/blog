package app

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// fakeArticleSource 返回预设的已发表文章分页数据。
type fakeArticleSource struct {
	articles []searchdomain.SourceArticle // 预设已发表文章
	err      error                        // 预设读取错误
}

// ListPublishedAfter 按 ID 游标返回预设文章。
func (f *fakeArticleSource) ListPublishedAfter(_ context.Context, lastID uint64, limit int) ([]searchdomain.SourceArticle, error) {
	// 1. 返回预设错误或游标后的文章
	if f.err != nil {
		return nil, f.err
	}
	result := make([]searchdomain.SourceArticle, 0, limit)
	for _, article := range f.articles {
		if article.ID <= lastID {
			continue
		}
		result = append(result, article)
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

// CountPublished 返回预设文章数量。
func (f *fakeArticleSource) CountPublished(context.Context) (uint64, error) {
	// 1. 返回预设文章数量
	return uint64(len(f.articles)), f.err
}

// rebuildWriter 记录全量重建执行的索引管理操作。
type rebuildWriter struct {
	created        []string                       // 已创建的物理索引
	documents      []searchdomain.ArticleDocument // 已写入的搜索文档
	deleted        []string                       // 已删除的失败索引
	alias          string                         // 已切换的稳定别名
	aliasIndex     string                         // 别名切换目标物理索引
	countOverride  *uint64                        // 覆盖索引数量，空时使用写入文档数
	createErr      error                          // 创建索引错误
	bulkErr        error                          // Bulk 写入错误
	switchAliasErr error                          // 别名切换错误
}

// CreateIndex 记录物理索引创建。
func (w *rebuildWriter) CreateIndex(_ context.Context, indexName string) error {
	// 1. 保存创建记录
	w.created = append(w.created, indexName)
	return w.createErr
}

// BulkUpsert 记录重建文档写入。
func (w *rebuildWriter) BulkUpsert(_ context.Context, _ string, documents []searchdomain.ArticleDocument) error {
	// 1. 保存文档写入记录
	w.documents = append(w.documents, documents...)
	return w.bulkErr
}

// DeleteDocuments 满足 IndexWriter 接口。
func (w *rebuildWriter) DeleteDocuments(context.Context, string, []uint64) error { return nil }

// Refresh 满足 IndexWriter 接口。
func (w *rebuildWriter) Refresh(context.Context, string) error { return nil }

// Count 返回实际写入数量或测试覆盖值。
func (w *rebuildWriter) Count(context.Context, string) (uint64, error) {
	// 1. 返回测试需要的索引数量
	if w.countOverride != nil {
		return *w.countOverride, nil
	}
	return uint64(len(w.documents)), nil
}

// SwitchAlias 记录稳定别名切换。
func (w *rebuildWriter) SwitchAlias(_ context.Context, alias, newIndex string) error {
	// 1. 保存别名切换记录
	w.alias = alias
	w.aliasIndex = newIndex
	return w.switchAliasErr
}

// DeleteIndex 记录失败物理索引清理。
func (w *rebuildWriter) DeleteIndex(_ context.Context, indexName string) error {
	// 1. 保存失败索引清理记录
	w.deleted = append(w.deleted, indexName)
	return nil
}

// TestRebuildServiceRebuild 验证分页构建、数量校验和别名切换。
func TestRebuildServiceRebuild(t *testing.T) {
	// 1. 准备三篇文章并使用两篇一批的分页大小
	source := &fakeArticleSource{articles: []searchdomain.SourceArticle{
		publishedArticle(1), publishedArticle(2), publishedArticle(3),
	}}
	writer := &rebuildWriter{}
	service := NewRebuildService(source, writer, NewDocumentFactory(fakeExtractor{content: "正文"}), "article_search", 2)
	service.now = func() time.Time { return time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC) }

	// 2. 执行重建并核对索引名、文档和别名
	indexName, err := service.Rebuild(context.Background())
	if err != nil {
		t.Fatalf("全量重建失败: %v", err)
	}
	if indexName != "article_search_20260827093000" || writer.alias != "article_search" || writer.aliasIndex != indexName {
		t.Fatalf("重建索引或别名不符合预期: index=%s alias=%s target=%s", indexName, writer.alias, writer.aliasIndex)
	}
	if len(writer.documents) != 3 || len(writer.deleted) != 0 {
		t.Fatalf("重建文档或清理记录不符合预期: documents=%d deleted=%v", len(writer.documents), writer.deleted)
	}
	if !reflect.DeepEqual(writer.created, []string{indexName}) {
		t.Fatalf("物理索引创建记录不符合预期: %v", writer.created)
	}
}

// TestRebuildServiceCleanupOnFailure 验证写入、数量和别名失败时清理新索引。
func TestRebuildServiceCleanupOnFailure(t *testing.T) {
	// 1. 定义三类重建失败场景
	wrongCount := uint64(0)
	tests := []struct {
		name   string         // 测试场景名称
		writer *rebuildWriter // 预设失败写入器
	}{
		{name: "Bulk 失败", writer: &rebuildWriter{bulkErr: errors.New("bulk 失败")}},
		{name: "数量不一致", writer: &rebuildWriter{countOverride: &wrongCount}},
		{name: "别名切换失败", writer: &rebuildWriter{switchAliasErr: errors.New("alias 失败")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewRebuildService(
				&fakeArticleSource{articles: []searchdomain.SourceArticle{publishedArticle(1)}},
				test.writer,
				NewDocumentFactory(fakeExtractor{content: "正文"}),
				"article_search",
				10,
			)
			service.now = func() time.Time { return time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC) }
			if _, err := service.Rebuild(context.Background()); err == nil {
				t.Fatal("重建失败场景未返回错误")
			}
			if !reflect.DeepEqual(test.writer.deleted, []string{"article_search_20260827093000"}) {
				t.Fatalf("失败新索引未被清理: %v", test.writer.deleted)
			}
		})
	}
}

// TestRebuildServiceEmptySource 验证空文章库仍能创建可用空索引。
func TestRebuildServiceEmptySource(t *testing.T) {
	// 1. 使用空文章数据源执行重建
	writer := &rebuildWriter{}
	service := NewRebuildService(&fakeArticleSource{}, writer, NewDocumentFactory(fakeExtractor{content: ""}), "article_search", 10)
	service.now = func() time.Time { return time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC) }

	// 2. 空索引数量校验后仍切换别名
	if _, err := service.Rebuild(context.Background()); err != nil {
		t.Fatalf("空文章库重建失败: %v", err)
	}
	if len(writer.documents) != 0 || writer.alias == "" {
		t.Fatalf("空文章库重建结果不符合预期: documents=%d alias=%s", len(writer.documents), writer.alias)
	}
}

// TestRebuildServiceDocumentConversionFailure 验证文档转换失败时保留错误链并清理新索引。
func TestRebuildServiceDocumentConversionFailure(t *testing.T) {
	// 1. 准备正文提取错误和单篇来源文章
	extractErr := errors.New("Markdown 解析失败")
	writer := &rebuildWriter{}
	service := NewRebuildService(
		&fakeArticleSource{articles: []searchdomain.SourceArticle{publishedArticle(1)}},
		writer,
		NewDocumentFactory(fakeExtractor{err: extractErr}),
		"article_search",
		10,
	)
	service.now = func() time.Time { return time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC) }

	// 2. 转换失败保留根因并删除失败物理索引
	_, err := service.Rebuild(context.Background())
	if !errors.Is(err, extractErr) || !reflect.DeepEqual(writer.deleted, []string{"article_search_20260827093000"}) {
		t.Fatalf("文档转换失败处理不符合预期: err=%v deleted=%v", err, writer.deleted)
	}
}
