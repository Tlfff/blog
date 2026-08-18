// Package trace 提供跨 HTTP/gRPC/Kafka 的 Trace ID 生成能力。
package trace

import "github.com/google/uuid"

// HeaderName 是 HTTP 与 gRPC metadata 中使用的 Trace ID 字段名。
const HeaderName = "X-Trace-ID"

// 生成一条请求或消息的 Trace ID
func NewID() string {
	return uuid.New().String()
}
