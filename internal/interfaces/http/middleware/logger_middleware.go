package middleware

import (
	"bytes"
	"io"
	"log"
	"time"

	"blog/shared/platform/trace"

	"github.com/gin-gonic/gin"
)

const maxBodyLogSize = 1024 // 最多打印 1KB 请求体

// LoggerMiddleware 链路追踪日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 接口进入前：生成或获取链路 TraceID
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = trace.NewID()
		}

		// 将 TraceID 注入上下文，方便后续的 Handler 或 Service 打印日志时获取
		c.Set("traceID", traceID)
		// 顺手让响应头也带上，方便前端拿着这个 ID 找后端对账
		c.Header("X-Trace-ID", traceID)

		startTime := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		authHeader := c.GetHeader("Authorization")

		// 2. 读取并打印请求体（POST）
		var bodyLog string
		if method == "POST" {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				bodyLog = "[读取请求体失败: " + err.Error() + "]"
			} else {
				// 必须重置 Body，否则后续 Handler 读不到数据
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				if len(bodyBytes) == 0 {
					bodyLog = "[空]"
				} else if len(bodyBytes) > maxBodyLogSize {
					bodyLog = string(bodyBytes[:maxBodyLogSize]) + "...[截断]"
				} else {
					bodyLog = string(bodyBytes)
				}
			}
		}

		log.Printf("[Trace-Start] ID: %s | %s | %s | IP: %s | Query: %s | Body: %s | Auth: %s | UA: %s",
			traceID, method, path, clientIP, query, bodyLog, authHeader, userAgent,
		)

		// 3. 执行核心业务逻辑
		c.Next()

		// 4. 接口退出后：收集状态、计算耗时并打印结束日志
		endTime := time.Now()
		latencyTime := endTime.Sub(startTime)
		statusCode := c.Writer.Status()

		log.Printf("[Trace-End] ID: %s | Status: %d | Latency: %v | Errors: %s",
			traceID, statusCode, latencyTime, c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}
