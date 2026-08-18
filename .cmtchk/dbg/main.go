package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

func norm(src string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, parser.SkipObjectResolution)
	if err != nil {
		return "ERR:" + err.Error()
	}
	f.Comments = nil
	f.Doc = nil
	ast.Inspect(f, func(n ast.Node) bool { return true })
	var buf bytes.Buffer
	(&printer.Config{Mode: printer.RawFormat}).Fprint(&buf, fset, f)
	return strings.Join(strings.Fields(buf.String()), " ")
}

func main() {
	fmt.Printf("OLD=%q\n", norm("package like\n"))
	fmt.Printf("NEW=%q\n", norm("// Package like 定义点赞相关的请求与响应 DTO。\npackage like\n"))
}
