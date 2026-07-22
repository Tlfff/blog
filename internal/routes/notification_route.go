package routes

import (
	"blog/internal/handler"

	"github.com/gin-gonic/gin"
)

// 通知路由
func InitNotificationPrivateRoutes(r *gin.RouterGroup, NotificationHandler *handler.NotificationHandler) {
	// 获取未读通知数量
	r.GET("/ntf/unread-count", NotificationHandler.GetUnreadCount)
	// 获取通知列表
	r.GET("/ntf/list", NotificationHandler.GetNotificationList)
	// 清空未读消息
	r.POST("/ntf/clear-unread", NotificationHandler.ClearUnread)
}
