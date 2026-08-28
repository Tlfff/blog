package infra

import (
	"reflect"
	"testing"
)

// TestMarkdownImageReferenceParserExtract 验证系统图片引用提取边界。
func TestMarkdownImageReferenceParserExtract(t *testing.T) {
	// 1. 使用表驱动用例覆盖有效引用、去重和忽略规则
	tests := []struct {
		name     string   // 测试场景名称
		markdown string   // 待解析 Markdown 正文
		expected []uint64 // 期望按出现顺序返回的图片 ID
	}{
		{name: "空正文", markdown: "", expected: []uint64{}},
		{name: "提取并去重", markdown: "![a](image://12)\n![b](image://7)\n![c](image://12)", expected: []uint64{12, 7}},
		{name: "忽略普通链接", markdown: "[link](image://12)", expected: []uint64{}},
		{name: "忽略代码", markdown: "`![inline](image://1)`\n```md\n![block](image://2)\n```", expected: []uint64{}},
		{name: "忽略历史与外部地址", markdown: "![old](https://cdn.example/a.png)\n![external](https://example.com/b.png)", expected: []uint64{}},
		{name: "忽略非法ID", markdown: "![zero](image://0) ![text](image://abc) ![path](image://1/a) ![query](image://2?x=1)", expected: []uint64{}},
	}
	parser := NewMarkdownImageReferenceParser()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// 1.1 解析正文并核对图片 ID 集合
			actual, err := parser.Extract(test.markdown)
			if err != nil {
				t.Fatalf("提取图片引用失败: %v", err)
			}
			if !reflect.DeepEqual(actual, test.expected) {
				t.Fatalf("图片引用不匹配: got=%v want=%v", actual, test.expected)
			}
		})
	}
}
