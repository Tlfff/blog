// Package log 提供统一的 Trace 日志辅助函数。
package log

import (
	"log"
	"runtime/debug"
)

// 记录错误日志，可选附带调用栈，err 为 nil 时不输出
func Error(prefix string, err error, withStack bool) {
	if err == nil {
		return
	}
	if withStack {
		log.Printf("%s: %v\n%s", prefix, err, debug.Stack())
		return
	}
	log.Printf("%s: %v", prefix, err)
}
