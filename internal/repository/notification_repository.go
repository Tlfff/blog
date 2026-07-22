package repository

import (
	"blog/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationRepository interface {
	Insert(ctx context.Context, notify *model.Notification) error
	GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*model.Notification, error)
	MarkAllAsRead(ctx context.Context, receiverID uint64) error
	GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error)
}

type notificationRepository struct {
	db *mongo.Database
}

func NewNotificationRepository(db *mongo.Database) NotificationRepository {
	return &notificationRepository{
		db: db,
	}
}

// 新增点赞通知
//
//	db.notifications.insertOne({
//	   "receiver_id": NumberLong(9527),
//	   "type": NumberInt(1),
//	   "is_read": false,
//	   "sender": { "user_id": 1001, "nickname": "xxx", "avatar": "xxx" },
//	   "content": { "article_id": 123, "article_title": "xxx" },
//	   "created_time": ISODate("2026-07-20T18:00:00Z")
//	})
func (r *notificationRepository) Insert(ctx context.Context, notify *model.Notification) error {
	_, err := r.db.Collection("notifications").InsertOne(ctx, notify)
	return err
}

// 获取用户的通知列表
// db.notifications.find({ "receiver_id": ？ })
//
//	.sort({ "created_time": -1 })
//	.skip(？)
//	.limit(？)
func (r *notificationRepository) GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*model.Notification, error) {

	var list []*model.Notification

	// 1. 查询条件
	filter := bson.M{"receiver_id": receiverID}

	// 2. 组装 MQL 中的 sort, skip, limit 条件
	opts := options.Find().
		SetSort(bson.M{"created_time": -1}). // 按创建时间倒序排列
		SetLimit(pageSize).                  // 限制每页条数
		SetSkip((page - 1) * pageSize)       // 计算跳过的条数

	// 3. 执行查询
	cursor, err := r.db.Collection("notifications").Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // 关闭游标，防止连接永久占用

	// 4. 将查出来的文档一把反序列化到 list 切片中
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}

	return list, nil
}

// 全量标记已读
// db.notifications.updateMany(
//
//	{ "receiver_id": ？, "is_read": false },
//	{ "$set": { "is_read": true } }
//
// )
func (r *notificationRepository) MarkAllAsRead(ctx context.Context, receiverID uint64) error {
	// 1. 组装过滤条件：属于当前用户，且目前状态还是“未读”的通知
	filter := bson.M{"receiver_id": receiverID, "is_read": false}

	// 2. 组装要更新的字段
	update := bson.M{"$set": bson.M{"is_read": true}}

	// 3. 执行批量更新
	_, err := r.db.Collection("notifications").UpdateMany(ctx, filter, update)
	return err
}

// 获取用户的未读消息数量
//
//	db.notification.countDocuments({
//		receiver_id:?
//		is_read:false
//	})
func (r *notificationRepository) GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error) {
	// 1. 过滤条件
	filter := bson.M{
		"receiver_id": receiverID,
		"is_read":     false,
	}
	// 2. 直接返回符合条件的条数
	count, err := r.db.Collection("notifications").CountDocuments(ctx, filter)
	return count, err
}
