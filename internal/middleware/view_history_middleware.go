package middleware

import (
	"blog/internal/auth"
	"blog/internal/dto/article"
	"blog/internal/service"
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// 浏览历史中间件：主流程成功返回后，异步发送浏览历史消息到 Kafka
func ViewHistoryMiddleware(historyService *service.ArticleViewHistoryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 解析查询参数中的文章ID，解析失败直接放行，由 handler 统一返回参数错误
		var req article.GetDetailRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			c.Next()
			return
		}

		// 2. 执行主流程
		c.Next()

		// 3. 主流程失败（参数错误、文章不存在、已删除、无权限等）时不记录浏览历史
		if len(c.Errors) > 0 {
			return
		}

		// 4. 从Gin 上下文获取用户信息，可能未登录
		userID := uint64(0)
		if userCtx, exists := c.Get("currentUser"); exists {
			userID = userCtx.(*auth.UserContext).UserID
		}

		// 5. 异步发送浏览历史消息，避免阻塞响应
		// 注意：不能使用 c.Request.Context()，请求结束后它会被取消，导致发送失败
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := historyService.SendViewHistory(ctx, userID, req.ID); err != nil {
				log.Printf("[Kafka] 中间件发送浏览历史失败, user: %d, article: %d, err: %v", userID, req.ID, err)
			}
		}()
	}
}
