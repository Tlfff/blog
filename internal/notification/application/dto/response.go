// Package notification 定义通知相关的请求与响应 DTO。
package notification

import (
	domain "blog/internal/notification/domain"
)

// NotifyListItem 是通知列表项响应 DTO。
type NotifyListItem struct {
	ID          string `json:"id"`           // 通知ID
	Type        int8   `json:"type"`         // 1:点赞文章, 2:点赞评论, 3:回复文章, 4:回复评论
	IsRead      bool   `json:"is_read"`      // 是否已读
	CreatedTime int64  `json:"created_time"` // 创建时间

	// 1. 动作发出者
	SenderID       uint64 `json:"sender_id"`       //用户ID
	SenderNickname string `json:"sender_nickname"` // 用户昵称
	SenderAvatar   string `json:"sender_avatar"`   // 用户头像

	// 2. 动作内容
	ActionText string `json:"action_text"` // 比如“赞了你的帖子”

	// 3. 操作文章
	ArticleID uint64 `json:"article_id"` // 文章 ID
	Title     string `json:"title"`      // 文章标题
}

// NotificationListResponse 是通知列表响应 DTO。
type NotificationListResponse struct {
	List     []*NotifyListItem `json:"list"`      // 通知列表
	Page     int64             `json:"page"`      // 当前页码
	PageSize int64             `json:"page_size"` // 每页大小
}

// 构造通知列表响应，将 MongoDB 通知文档按类型映射为前端展示字段
// 构造列表响应函数
func NewNotificationListResponse(notifications []*domain.Notification, page, pageSize int64) *NotificationListResponse {
	resp := &NotificationListResponse{
		List:     make([]*NotifyListItem, 0),
		Page:     page,
		PageSize: pageSize,
	}

	for _, notification := range notifications {
		// 1. 组装与通知类型无关的通用字段
		item := &NotifyListItem{
			ID:             notification.ID,
			Type:           notification.Type,
			IsRead:         notification.IsRead,
			CreatedTime:    notification.CreatedTime.Unix(),
			SenderID:       notification.Sender.UserID,
			SenderNickname: notification.Sender.Nickname,
			SenderAvatar:   notification.Sender.Avatar,
		}

		// 2. 按通知类型映射动作文案与关联文章信息
		switch notification.Type {
		case domain.NotifyTypeLikeArticle:
			content, _ := notification.Content.(domain.LikeArticleContent)
			item.ActionText = "赞了你的文章"
			item.ArticleID = content.ArticleID
			item.Title = content.ArticleTitle
		case domain.NotifyTypeLikeComment:
			item.ActionText = "赞了你的评论"
		case domain.NotifyTypeCommentArticle:
			content, _ := notification.Content.(domain.CommentContent)
			item.ActionText = "评论了你的文章"
			item.ArticleID = content.ArticleID
		case domain.NotifyTypeReplyComment:
			content, _ := notification.Content.(domain.CommentContent)
			item.ActionText = "回复了你的评论"
			item.ArticleID = content.ArticleID
		default:
			item.ActionText = "收到一条新消息"
		}

		// 3. 追加到列表
		resp.List = append(resp.List, item)
	}
	return resp
}
