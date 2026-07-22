package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/notification"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notifyService *service.NotificationService
}

func NewNotificationHandler(notifyService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notifyService: notifyService,
	}
}

// 获取通知列表（进入消息中心时前端调用）
func (h *NotificationHandler) GetNotificationList(c *gin.Context) {
	var req notification.NotifyListRequest
	// 1. 解析分页参数（给好默认值）
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrParameter)
		return
	}

	// 2. 从上下文中获取用户信息
	user := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 获取通知数据
	resp, err := h.notifyService.GetMyNotifications(c.Request.Context(), user.UserID, int64(req.Page), int64(req.PageSize))
	if err != nil {
		c.Error(err)
		return
	}

	// 4. 返回结果
	common.OK(c, "获取通知列表成功", resp)
}

// 一键清除未读红点
func (h *NotificationHandler) ClearUnread(c *gin.Context) {
	// 1. 从上下文中获取用户信息
	user := c.MustGet("currentUser").(*auth.UserContext)

	// 2. 调用 Service 清除未读状态
	err := h.notifyService.ClearUnread(c.Request.Context(), user.UserID)
	if err != nil {
		c.Error(err)
		return
	}
	// 3. 返回结果
	common.OK(c, "消除未读消息成功", nil)
}

// 获取未读消息数
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	// 1. 从上下文中获取用户信息
	user := c.MustGet("currentUser").(*auth.UserContext)

	// 2. 获取未读消息数量
	count, err := h.notifyService.GetUnreadCount(c.Request.Context(), user.UserID)
	if err != nil {
		c.Error(err)
		return
	}
	// 3. 返回结果
	common.OK(c, "获取未读消息数量成功", count)
}
