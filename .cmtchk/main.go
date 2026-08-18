package main

// 工具：校验相对 git HEAD 的改动是否仅为注释改动（去注释后 AST 一致）
import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"strings"
)

func normalize(src []byte) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, parser.SkipObjectResolution)
	if err != nil {
		return "", err
	}
	f.Comments = nil
	ast.Inspect(f, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.GenDecl:
			t.Doc = nil
		case *ast.FuncDecl:
			t.Doc = nil
		case *ast.Field:
			t.Doc, t.Comment = nil, nil
		case *ast.TypeSpec:
			t.Doc, t.Comment = nil, nil
		case *ast.ValueSpec:
			t.Doc, t.Comment = nil, nil
		case *ast.ImportSpec:
			t.Doc, t.Comment = nil, nil
		case *ast.File:
			t.Doc = nil
		}
		return true
	})
	var buf bytes.Buffer
	if err := (&printer.Config{Mode: printer.RawFormat}).Fprint(&buf, fset, f); err != nil {
		return "", err
	}
	return strings.Join(strings.Fields(buf.String()), " "), nil
}

func main() {
	out, err := exec.Command("git", "diff", "--name-only", "HEAD", "--", "*.go").Output()
	if err != nil {
		fmt.Println("git error:", err)
		os.Exit(1)
	}
	bad := 0
	n := 0
	for _, p := range strings.Fields(string(out)) {
		if strings.HasPrefix(p, ".cmtchk/") {
			continue
		}
		cur, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		old, err := exec.Command("git", "show", "HEAD:"+p).Output()
		if err != nil {
			continue
		}
		a, e1 := normalize(old)
		b, e2 := normalize(cur)
		if e1 != nil || e2 != nil {
			fmt.Println("PARSE-FAIL", p, e1, e2)
			bad++
			continue
		}
		n++
		if a != b {
			fmt.Println("CODE-CHANGED!", p)
			bad++
		}
	}
	fmt.Printf("checked=%d mismatches=%d\n", n, bad)
	if bad > 0 {
		os.Exit(1)
	}
	fmt.Println("ALL-COMMENT-ONLY")
}
