package notification

import (
	"blog/internal/model"
	"log"

	"go.mongodb.org/mongo-driver/bson"
)

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
type NotificationListResponse struct {
	List     []*NotifyListItem `json:"list"`
	Page     int64             `json:"page"`      // 当前页码
	PageSize int64             `json:"page_size"` // 每页大小
}

// 构造列表响应函数
func NewNotificationListResponse(models []*model.Notification, page, pageSize int64) *NotificationListResponse {
	resp := &NotificationListResponse{
		List:     make([]*NotifyListItem, 0),
		Page:     page,
		PageSize: pageSize,
	}

	for _, m := range models {
		// 构建通用的字段
		item := &NotifyListItem{
			ID:             m.ID.Hex(), // MongoDB 的 Primitive.ObjectID 转换为 Hex 字符串
			Type:           m.Type,
			IsRead:         m.IsRead,
			CreatedTime:    m.CreatedTime.Unix(),
			SenderID:       m.Sender.UserID,
			SenderNickname: m.Sender.Nickname,
			SenderAvatar:   m.Sender.Avatar,
		}

		// 根据不同的通知类型，映射具体的动作和文章内容
		switch m.Type {
		case model.NotifyTypeLikeArticle: // 点赞文章
			var content model.LikeArticleNotifyContent

			// 将动态内容转码赋值给具体的结构体
			if bytes, err := bson.Marshal(m.Content); err == nil {
				_ = bson.Unmarshal(bytes, &content)
			}
			item.ActionText = "赞了你的文章"
			item.ArticleID = content.ArticleID
			item.Title = content.ArticleTitle
		default: // 兜底，当收到一个未定义的消息
			item.ActionText = "收到一条新消息"
			log.Printf("[WARN] 未知的通知类型: type=%d, notify_id=%s", m.Type, m.ID.Hex())
		}

		resp.List = append(resp.List, item)
	}
	return resp
}
