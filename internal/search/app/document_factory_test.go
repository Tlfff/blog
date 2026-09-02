package app

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// fakeExtractor 返回预设正文或错误。
type fakeExtractor struct {
	content string // 预设纯文本正文
	err     error  // 预设提取错误
}

// Extract 返回预设结果。
func (f fakeExtractor) Extract(context.Context, string) (string, error) {
	// 1. 返回测试预设结果
	return f.content, f.err
}

// TestNormalizeTags 验证标签去空白、空值和大小写重复项。
func TestNormalizeTags(t *testing.T) {
	// 1. 规范化混合标签并保持首次出现的展示值
	result := NormalizeTags(" Go, go , ,Web,WEB, 中文 ")
	expected := []string{"Go", "Web", "中文"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("标签规范化结果不符合预期: got=%v want=%v", result, expected)
	}
}

// TestDocumentFactoryBuild 验证统一搜索文档转换。
func TestDocumentFactoryBuild(t *testing.T) {
	// 1. 准备来源文章和纯文本提取结果
	updatedTime := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	factory := NewDocumentFactory(fakeExtractor{content: "纯文本正文"})

	// 2. 构建文档并核对字段
	document, err := factory.Build(context.Background(), searchdomain.SourceArticle{
		ID: 3, Title: "搜索标题", Content: "**正文**", Tags: "Go, go,ES", UpdatedTime: updatedTime,
	})
	if err != nil {
		t.Fatalf("构建搜索文档失败: %v", err)
	}
	if document.ArticleID != 3 || document.Content != "纯文本正文" || !reflect.DeepEqual(document.Tags, []string{"Go", "ES"}) {
		t.Fatalf("搜索文档不符合预期: %+v", document)
	}
}

// TestDocumentFactoryPreservesError 验证正文提取错误链和文章 ID。
func TestDocumentFactoryPreservesError(t *testing.T) {
	// 1. 准备可识别的底层错误
	extractErr := errors.New("解析失败")
	factory := NewDocumentFactory(fakeExtractor{err: extractErr})

	// 2. 构建失败时保留底层错误链
	_, err := factory.Build(context.Background(), searchdomain.SourceArticle{ID: 9})
	if !errors.Is(err, extractErr) {
		t.Fatalf("正文提取错误链丢失: %v", err)
	}
}
