package app

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"fmt"
	"strings"
)

// DocumentFactory 把来源文章转换为统一搜索文档。
type DocumentFactory struct {
	extractor searchdomain.TextExtractor // Markdown 纯文本提取器
}

// NewDocumentFactory 创建搜索文档工厂。
func NewDocumentFactory(extractor searchdomain.TextExtractor) *DocumentFactory {
	// 1. 保存正文提取依赖
	return &DocumentFactory{extractor: extractor}
}

// Build 根据来源文章构建可写入 Elasticsearch 的搜索文档。
func (f *DocumentFactory) Build(ctx context.Context, article searchdomain.SourceArticle) (searchdomain.ArticleDocument, error) {
	// 1. 提取正文普通文本
	if f == nil || f.extractor == nil {
		return searchdomain.ArticleDocument{}, fmt.Errorf("文章 %d 缺少 Markdown 文本提取器", article.ID)
	}
	content, err := f.extractor.Extract(ctx, article.Content)
	if err != nil {
		return searchdomain.ArticleDocument{}, fmt.Errorf("提取文章 %d 正文纯文本失败: %w", article.ID, err)
	}

	// 2. 规范化标签并组装稳定搜索文档
	return searchdomain.ArticleDocument{
		ArticleID:   article.ID,
		Title:       article.Title,
		Content:     content,
		Tags:        NormalizeTags(article.Tags),
		UpdatedTime: article.UpdatedTime,
	}, nil
}

// NormalizeTags 规范化英文逗号分隔的文章标签。
func NormalizeTags(rawTags string) []string {
	// 1. 按原始顺序去除空白、空标签和英文大小写重复项
	normalized := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawTag := range strings.Split(rawTags, ",") {
		tag := strings.TrimSpace(rawTag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized
}
