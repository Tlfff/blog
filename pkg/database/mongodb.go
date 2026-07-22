package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 负责连接 MongoDB
func NewMongoDBClient(username, password, host, dbname string, port int) (*mongo.Database, error) {
	var dsn string
	dsn = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", username, password, host, port, dbname)

	// 1. 设置连接超时控制（防止数据库挂了导致程序永久死等）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. 建立连接
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, err
	}

	// 3. 顺着管道给 MongoDB 发个 Ping，确保网络和认证是切实可通的
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client.Database(dbname), nil
}
