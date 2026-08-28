package infra

import (
	"context"
	"strings"
	"testing"
)

// TestMarkdownExtractorExtract 验证 Markdown 纯文本提取边界。
func TestMarkdownExtractorExtract(t *testing.T) {
	// 1. 定义正常文本、链接、图片和代码混合场景
	markdown := `# Canal 同步

使用 **Canal** 监听 [MySQL](https://mysql.com)，并支持 *全文搜索*。

裸地址 https://example.com/article 也不参与搜索。

![架构图](https://example.com/diagram.png)

行内代码 ` + "`context.Context`" + ` 不参与搜索。

~~~go
func main() {}
~~~

> 保留引用文字

- 保留列表文字
`
	extractor := NewMarkdownExtractor()

	// 2. 提取结果保留普通文本并排除地址、图片和代码
	result, err := extractor.Extract(context.Background(), markdown)
	if err != nil {
		t.Fatalf("提取 Markdown 失败: %v", err)
	}
	for _, expected := range []string{"Canal 同步", "使用", "Canal", "监听", "MySQL", "全文搜索", "保留引用文字", "保留列表文字"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("提取结果缺少普通文本 %q: %q", expected, result)
		}
	}
	for _, excluded := range []string{"https://", "架构图", "context.Context", "func main"} {
		if strings.Contains(result, excluded) {
			t.Fatalf("提取结果包含应排除内容 %q: %q", excluded, result)
		}
	}
}

// TestMarkdownExtractorEmptyAndCanceled 验证空正文和取消上下文。
func TestMarkdownExtractorEmptyAndCanceled(t *testing.T) {
	// 1. 空正文返回空文本
	extractor := NewMarkdownExtractor()
	result, err := extractor.Extract(context.Background(), "   ")
	if err != nil || result != "" {
		t.Fatalf("空正文提取结果不符合预期: result=%q err=%v", result, err)
	}

	// 2. 已取消上下文停止解析
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := extractor.Extract(ctx, "正文"); err == nil {
		t.Fatal("已取消上下文未返回错误")
	}
}
