package infrastructure

import (
	notificationdomain "blog/internal/notification/domain"
	"blog/internal/notification/infrastructure/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// notificationRepository 是通知 Repository 的 MongoDB 实现。
type notificationRepository struct {
	db *mongo.Database // MongoDB 数据库句柄
}

// NewNotificationRepository 创建通知 Repository。
func NewNotificationRepository(db *mongo.Database) notificationdomain.NotificationRepository {
	return &notificationRepository{db: db}
}

// Insert 写入一条通知，并回填 MongoDB ObjectID。
func (r *notificationRepository) Insert(ctx context.Context, notification *notificationdomain.Notification) error {
	// 1. 将领域对象转换为 MongoDB 文档
	document := toModelNotification(notification)

	// 2. 写入 notifications 集合
	result, err := r.db.Collection("notifications").InsertOne(ctx, document)
	if err != nil {
		return err
	}

	// 3. 回填 ObjectID 的十六进制字符串
	if id, ok := result.InsertedID.(primitive.ObjectID); ok {
		notification.ID = id.Hex()
	}
	return nil
}

// GetList 按接收者和创建时间倒序分页查询通知。
func (r *notificationRepository) GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*notificationdomain.Notification, error) {
	// 1. 组装分页和排序条件
	opts := options.Find().
		SetSort(bson.M{"created_time": -1}).
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize)

	// 2. 查询接收者的通知文档
	cursor, err := r.db.Collection("notifications").Find(ctx, bson.M{"receiver_id": receiverID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 3. 解析并转换为领域对象
	var documents []*model.Notification
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}
	list := make([]*notificationdomain.Notification, 0, len(documents))
	for _, document := range documents {
		list = append(list, toDomainNotification(document))
	}
	return list, nil
}

// MarkAllAsRead 将接收者全部未读通知标记为已读。
func (r *notificationRepository) MarkAllAsRead(ctx context.Context, receiverID uint64) error {
	_, err := r.db.Collection("notifications").UpdateMany(ctx,
		bson.M{"receiver_id": receiverID, "is_read": false},
		bson.M{"$set": bson.M{"is_read": true}},
	)
	return err
}

// GetUnreadCount 查询接收者的未读通知数量。
func (r *notificationRepository) GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error) {
	return r.db.Collection("notifications").CountDocuments(ctx, bson.M{
		"receiver_id": receiverID,
		"is_read":     false,
	})
}

// toModelNotification 将领域对象转换为 MongoDB 文档。
func toModelNotification(notification *notificationdomain.Notification) *model.Notification {
	content := notification.Content
	if value, ok := notification.Content.(notificationdomain.LikeArticleContent); ok {
		content = model.LikeArticleNotifyContent{
			ArticleID:    value.ArticleID,
			ArticleTitle: value.ArticleTitle,
		}
	}
	return &model.Notification{
		ReceiverID: notification.ReceiverID,
		Sender: model.NotifySender{
			UserID:   notification.Sender.UserID,
			Nickname: notification.Sender.Nickname,
			Avatar:   notification.Sender.Avatar,
		},
		Type:        notification.Type,
		IsRead:      notification.IsRead,
		Content:     content,
		CreatedTime: notification.CreatedTime,
	}
}

// toDomainNotification 将 MongoDB 文档转换为领域对象。
func toDomainNotification(document *model.Notification) *notificationdomain.Notification {
	content := document.Content
	if document.Type == notificationdomain.NotifyTypeLikeArticle {
		var value model.LikeArticleNotifyContent
		if bytes, err := bson.Marshal(document.Content); err == nil {
			_ = bson.Unmarshal(bytes, &value)
		}
		content = notificationdomain.LikeArticleContent{
			ArticleID:    value.ArticleID,
			ArticleTitle: value.ArticleTitle,
		}
	}
	return &notificationdomain.Notification{
		ID:         document.ID.Hex(),
		ReceiverID: document.ReceiverID,
		Sender: notificationdomain.NotifySender{
			UserID:   document.Sender.UserID,
			Nickname: document.Sender.Nickname,
			Avatar:   document.Sender.Avatar,
		},
		Type:        document.Type,
		IsRead:      document.IsRead,
		Content:     content,
		CreatedTime: document.CreatedTime,
	}
}
