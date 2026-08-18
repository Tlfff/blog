// Package log 提供统一的 Trace 日志辅助函数。
package log

import (
	"log"
	"runtime/debug"
)

// Error 记录错误并附带调用栈开关。
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
