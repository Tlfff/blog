package infra

import (
	domaincontent "blog/internal/article/domain"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

const articleImageReferencePrefix = "image://"

// markdownImageReferenceParser 使用 Markdown AST 提取系统图片引用。
type markdownImageReferenceParser struct {
	markdown goldmark.Markdown // Goldmark Markdown 解析器
}

// NewMarkdownImageReferenceParser 创建文章图片引用解析器。
func NewMarkdownImageReferenceParser() domaincontent.ArticleImageReferenceParser {
	// 1. 使用项目现有 Goldmark 依赖创建解析器
	return &markdownImageReferenceParser{markdown: goldmark.New()}
}

// Extract 从 Markdown 图片节点中提取去重后的系统图片 ID。
func (p *markdownImageReferenceParser) Extract(markdown string) ([]uint64, error) {
	// 1. 空正文直接返回空集合
	if strings.TrimSpace(markdown) == "" {
		return []uint64{}, nil
	}

	// 2. 遍历图片节点并按正文出现顺序收集合法引用
	source := []byte(markdown)
	root := p.markdown.Parser().Parse(text.NewReader(source))
	ids := make([]uint64, 0)
	seen := make(map[uint64]struct{})
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		image, ok := node.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}
		imageID, ok := parseArticleImageReference(string(image.Destination))
		if !ok {
			return ast.WalkContinue, nil
		}
		if _, exists := seen[imageID]; exists {
			return ast.WalkContinue, nil
		}
		seen[imageID] = struct{}{}
		ids = append(ids, imageID)
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// parseArticleImageReference 解析严格格式的系统图片引用。
func parseArticleImageReference(reference string) (uint64, bool) {
	// 1. 校验协议前缀并解析正整数图片 ID
	if !strings.HasPrefix(reference, articleImageReferencePrefix) {
		return 0, false
	}
	value := strings.TrimPrefix(reference, articleImageReferencePrefix)
	if value == "" || strings.ContainsAny(value, "/?#") {
		return 0, false
	}
	imageID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || imageID == 0 {
		return 0, false
	}
	return imageID, true
}
