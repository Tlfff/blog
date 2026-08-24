package http

import (
	notificationresult "blog/internal/notification/app/dto"
	"blog/internal/platform/interfaces/http/response"
	"blog/internal/platform/security"
	apperrors "blog/internal/shared/apperrors"
	"context"

	"github.com/gin-gonic/gin"
)

// NotificationUsecase 是通知应用用例接口。
type NotificationUsecase interface {
	// GetMyNotifications 查询当前用户通知列表。
	GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int64) (*notificationresult.NotificationListResponse, error)
	// ClearUnread 将当前用户全部未读通知标记为已读。
	ClearUnread(ctx context.Context, userID uint64) error
	// GetUnreadCount 查询当前用户未读通知数量。
	GetUnreadCount(ctx context.Context, userID uint64) (int64, error)
}

// NotificationHandler 处理 Notification 上下文的 HTTP 请求。
type NotificationHandler struct {
	notifyService NotificationUsecase // 通知应用用例
}

// NewNotificationHandler 创建 Notification HTTP Handler。
func NewNotificationHandler(notifyService NotificationUsecase) *NotificationHandler {
	return &NotificationHandler{
		notifyService: notifyService,
	}
}

// GetNotificationList 获取当前用户通知列表。
func (h *NotificationHandler) GetNotificationList(c *gin.Context) {
	var req NotifyListRequest
	// 1. 解析分页参数（给好默认值）
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrParameter)
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
	response.OK(c, "获取通知列表成功", resp)
}

// ClearUnread 将当前用户全部未读通知标记为已读。
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
	response.OK(c, "消除未读消息成功", nil)
}

// GetUnreadCount 获取当前用户未读通知数量。
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
	response.OK(c, "获取未读消息数量成功", count)
}
