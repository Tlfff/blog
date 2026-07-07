package middleware

import (
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LoggerMiddleware 链路追踪日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 接口进入前：生成或获取链路 TraceID
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			// 如果上游没传，我们自己生成一个 UUID 丢进去
			traceID = uuid.New().String()
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

		log.Printf("[Trace-Start] ID: %s | %s | %s | IP: %s | Query: %s",
			traceID, method, path, clientIP, query,
		)

		// 2. 执行核心业务逻辑
		c.Next()

		// 3. 接口退出后：收集状态、计算耗时并打印结束日志
		endTime := time.Now()
		latencyTime := endTime.Sub(startTime) // 计算耗时
		statusCode := c.Writer.Status()       // 状态码 (200, 400, 500等)

		log.Printf("[Trace-End] ID: %s | Status: %d | Latency: %v | Errors: %s",
			traceID, statusCode, latencyTime, c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}
