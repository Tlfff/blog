package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// notificationRepository 是通知 Repository 的 MongoDB 实现。
type notificationRepository struct {
	db *mongo.Database // MongoDB 数据库句柄
}

// NewNotificationRepository 返回直接持有 MongoDB 的通知 Repository 实现。
func NewNotificationRepository(db *mongo.Database) domaincommunity.NotificationRepository {
	return &notificationRepository{db: db}
}

// 写入一条通知，并把生成的文档ID回填到领域对象
func (r *notificationRepository) Insert(ctx context.Context, notification *domaincommunity.Notification) error {
	// 1. 领域对象转换为 Mongo 文档模型
	m := toModelNotification(notification)
	// 2. 插入通知集合
	res, err := r.db.Collection("notifications").InsertOne(ctx, m)
	if err != nil {
		return err
	}
	// 3. 回填 MongoDB 生成的 ObjectID
	notification.ID = res.InsertedID.(interface{ Hex() string }).Hex()
	return nil
}

// 按接收者分页查询通知列表，按创建时间倒序
func (r *notificationRepository) GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*domaincommunity.Notification, error) {
	// 1. 按创建时间倒序并设置分页参数
	opts := options.Find().
		SetSort(bson.M{"created_time": -1}).
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize)
	// 2. 执行查询
	cursor, err := r.db.Collection("notifications").Find(ctx, bson.M{"receiver_id": receiverID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 3. 解析全部文档
	var models []*model.Notification
	if err := cursor.All(ctx, &models); err != nil {
		return nil, err
	}
	// 4. 逐条转换为领域对象
	list := make([]*domaincommunity.Notification, 0, len(models))
	for _, m := range models {
		list = append(list, toDomainNotification(m))
	}
	return list, nil
}

// 把接收者的全部未读通知标记为已读
func (r *notificationRepository) MarkAllAsRead(ctx context.Context, receiverID uint64) error {
	_, err := r.db.Collection("notifications").UpdateMany(ctx,
		bson.M{"receiver_id": receiverID, "is_read": false},
		bson.M{"$set": bson.M{"is_read": true}},
	)
	return err
}

// 统计接收者的未读通知数量
func (r *notificationRepository) GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error) {
	return r.db.Collection("notifications").CountDocuments(ctx, bson.M{
		"receiver_id": receiverID,
		"is_read":     false,
	})
}

// 把通知领域对象转换为 MongoDB 数据模型
func toModelNotification(n *domaincommunity.Notification) *model.Notification {
	// 1. 组装发送方公开信息
	sender := model.NotifySender{
		UserID:   n.Sender.UserID,
		Nickname: n.Sender.Nickname,
		Avatar:   n.Sender.Avatar,
	}
	// 2. 按通知类型转换动态内容
	var content any
	if like, ok := n.Content.(domaincommunity.LikeArticleContent); ok {
		content = model.LikeArticleNotifyContent{
			ArticleID:    like.ArticleID,
			ArticleTitle: like.ArticleTitle,
		}
	}
	// 3. 组装 Mongo 文档模型
	return &model.Notification{
		ReceiverID:  n.ReceiverID,
		Sender:      sender,
		Type:        n.Type,
		IsRead:      n.IsRead,
		Content:     content,
		CreatedTime: n.CreatedTime,
	}
}

// 把 MongoDB 数据模型转换为通知领域对象
func toDomainNotification(m *model.Notification) *domaincommunity.Notification {
	// 1. 组装通知基础字段
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
	// 2. 按类型还原动态内容
	if content, ok := m.Content.(model.LikeArticleNotifyContent); ok {
		n.Content = domaincommunity.LikeArticleContent{
			ArticleID:    content.ArticleID,
			ArticleTitle: content.ArticleTitle,
		}
	}
	return n
}
