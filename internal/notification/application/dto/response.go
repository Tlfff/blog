package notification

import (
	"blog/internal/notification/domain"
	"log"
)

// NotifyListItem 表示通知列表中的单条响应。
type NotifyListItem struct {
	ID          string `json:"id"`           // 通知唯一标识
	Type        int8   `json:"type"`         // 通知类型：1-点赞文章；2-点赞评论；3-评论文章；4-回复评论
	IsRead      bool   `json:"is_read"`      // 是否已读
	CreatedTime int64  `json:"created_time"` // 创建时间，Unix 秒

	SenderID       uint64 `json:"sender_id"`       // 通知发送方用户唯一标识
	SenderNickname string `json:"sender_nickname"` // 通知发送方昵称
	SenderAvatar   string `json:"sender_avatar"`   // 通知发送方头像

	ActionText string `json:"action_text"` // 通知动作描述
	ArticleID  uint64 `json:"article_id"`  // 关联文章唯一标识
	Title      string `json:"title"`       // 关联文章标题
}

// NotificationListResponse 表示通知分页响应。
type NotificationListResponse struct {
	List     []*NotifyListItem `json:"list"`      // 通知列表
	Page     int64             `json:"page"`      // 当前页码
	PageSize int64             `json:"page_size"` // 每页数量
}

// NewNotificationListResponse 将通知领域对象转换为兼容响应。
func NewNotificationListResponse(notifications []*domain.Notification, page, pageSize int64) *NotificationListResponse {
	response := &NotificationListResponse{
		List:     make([]*NotifyListItem, 0, len(notifications)),
		Page:     page,
		PageSize: pageSize,
	}
	for _, notification := range notifications {
		item := &NotifyListItem{
			ID:             notification.ID,
			Type:           notification.Type,
			IsRead:         notification.IsRead,
			CreatedTime:    notification.CreatedTime.Unix(),
			SenderID:       notification.Sender.UserID,
			SenderNickname: notification.Sender.Nickname,
			SenderAvatar:   notification.Sender.Avatar,
		}
		switch notification.Type {
		case domain.NotifyTypeLikeArticle:
			content, _ := notification.Content.(domain.LikeArticleContent)
			item.ActionText = "赞了你的文章"
			item.ArticleID = content.ArticleID
			item.Title = content.ArticleTitle
		default:
			item.ActionText = "收到一条新消息"
			log.Printf("[WARN] 未知的通知类型: type=%d, notify_id=%s", notification.Type, notification.ID)
		}
		response.List = append(response.List, item)
	}
	return response
}
