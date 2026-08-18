package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type notificationRepository struct {
	db *mongo.Database
}

// NewNotificationRepository 返回直接持有 MongoDB 的通知 Repository 实现。
func NewNotificationRepository(db *mongo.Database) domaincommunity.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Insert(ctx context.Context, notification *domaincommunity.Notification) error {
	m := toModelNotification(notification)
	res, err := r.db.Collection("notifications").InsertOne(ctx, m)
	if err != nil {
		return err
	}
	notification.ID = res.InsertedID.(interface{ Hex() string }).Hex()
	return nil
}

func (r *notificationRepository) GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*domaincommunity.Notification, error) {
	opts := options.Find().
		SetSort(bson.M{"created_time": -1}).
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize)
	cursor, err := r.db.Collection("notifications").Find(ctx, bson.M{"receiver_id": receiverID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var models []*model.Notification
	if err := cursor.All(ctx, &models); err != nil {
		return nil, err
	}
	list := make([]*domaincommunity.Notification, 0, len(models))
	for _, m := range models {
		list = append(list, toDomainNotification(m))
	}
	return list, nil
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, receiverID uint64) error {
	_, err := r.db.Collection("notifications").UpdateMany(ctx,
		bson.M{"receiver_id": receiverID, "is_read": false},
		bson.M{"$set": bson.M{"is_read": true}},
	)
	return err
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error) {
	return r.db.Collection("notifications").CountDocuments(ctx, bson.M{
		"receiver_id": receiverID,
		"is_read":     false,
	})
}

func toModelNotification(n *domaincommunity.Notification) *model.Notification {
	sender := model.NotifySender{
		UserID:   n.Sender.UserID,
		Nickname: n.Sender.Nickname,
		Avatar:   n.Sender.Avatar,
	}
	var content any
	if like, ok := n.Content.(domaincommunity.LikeArticleContent); ok {
		content = model.LikeArticleNotifyContent{
			ArticleID:    like.ArticleID,
			ArticleTitle: like.ArticleTitle,
		}
	}
	return &model.Notification{
		ReceiverID:  n.ReceiverID,
		Sender:      sender,
		Type:        n.Type,
		IsRead:      n.IsRead,
		Content:     content,
		CreatedTime: n.CreatedTime,
	}
}

func toDomainNotification(m *model.Notification) *domaincommunity.Notification {
	n := &domaincommunity.Notification{
		ID:         m.ID.Hex(),
		ReceiverID: m.ReceiverID,
		Sender: domaincommunity.NotifySender{
			UserID:   m.Sender.UserID,
			Nickname: m.Sender.Nickname,
			Avatar:   m.Sender.Avatar,
		},
		Type:        m.Type,
		IsRead:      m.IsRead,
		CreatedTime: m.CreatedTime,
	}
	if content, ok := m.Content.(model.LikeArticleNotifyContent); ok {
		n.Content = domaincommunity.LikeArticleContent{
			ArticleID:    content.ArticleID,
			ArticleTitle: content.ArticleTitle,
		}
	}
	return n
}
