package infra

import (
	"bytes"
	"context"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownExtractor 使用 Markdown AST 提取可搜索普通文本。
type MarkdownExtractor struct {
	parser goldmark.Markdown // Goldmark Markdown 解析器
}

// NewMarkdownExtractor 创建 Markdown 纯文本提取器。
func NewMarkdownExtractor() *MarkdownExtractor {
	// 1. 使用标准 Goldmark 解析器保留稳定 Markdown 语义
	return &MarkdownExtractor{parser: goldmark.New()}
}

// Extract 提取正文普通文本并排除图片、地址和代码节点。
func (e *MarkdownExtractor) Extract(ctx context.Context, markdown string) (string, error) {
	// 1. 上下文已取消时不继续解析正文
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(markdown) == "" {
		return "", nil
	}

	// 2. 解析 AST 并只收集允许搜索的文本节点
	source := []byte(markdown)
	document := e.parser.Parser().Parse(text.NewReader(source))
	var output bytes.Buffer
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if err := ctx.Err(); err != nil {
			return ast.WalkStop, err
		}
		if entering && shouldSkipNode(node) {
			return ast.WalkSkipChildren, nil
		}
		if entering {
			if textNode, ok := node.(*ast.Text); ok {
				appendText(&output, textNode.Segment.Value(source))
			}
			if stringNode, ok := node.(*ast.String); ok {
				appendText(&output, stringNode.Value)
			}
		}
		if !entering && isBlockBoundary(node) {
			appendText(&output, []byte(" "))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return "", err
	}

	// 3. 归一化连续空白后返回可搜索纯文本
	return normalizeSearchText(output.String()), nil
}

// normalizeSearchText 归一化正文空白并排除裸 URL。
func normalizeSearchText(content string) string {
	// 1. 逐词过滤 HTTP、HTTPS 和 www 裸地址
	words := strings.Fields(content)
	filtered := make([]string, 0, len(words))
	for _, word := range words {
		trimmed := strings.TrimLeft(word, "([{'\"《【")
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "www.") {
			continue
		}
		filtered = append(filtered, word)
	}
	return strings.Join(filtered, " ")
}

// shouldSkipNode 判断节点及其子节点是否应排除搜索。
func shouldSkipNode(node ast.Node) bool {
	// 1. 图片、代码和原始 HTML 均不进入正文搜索
	switch node.Kind() {
	case ast.KindImage, ast.KindCodeSpan, ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindRawHTML:
		return true
	default:
		return false
	}
}

// isBlockBoundary 判断节点结束时是否需要分隔相邻文本。
func isBlockBoundary(node ast.Node) bool {
	// 1. 块级内容结束后插入空格，避免不同段落错误粘连
	switch node.Kind() {
	case ast.KindParagraph, ast.KindHeading, ast.KindBlockquote, ast.KindListItem:
		return true
	default:
		return false
	}
}

// appendText 追加非空文本，并在单词边界需要时插入空格。
func appendText(output *bytes.Buffer, value []byte) {
	// 1. 空文本不影响最终结果
	if len(value) == 0 {
		return
	}
	if output.Len() > 0 {
		previous := rune(output.Bytes()[output.Len()-1])
		current := rune(value[0])
		if !unicode.IsSpace(previous) && !unicode.IsSpace(current) {
			output.WriteByte(' ')
		}
	}
	output.Write(value)
}
