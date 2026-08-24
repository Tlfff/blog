package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoDBClient 创建并检查 MongoDB 数据库连接。
//
// 参数说明：
//   - username：MongoDB 用户名。
//   - password：MongoDB 密码。
//   - host：MongoDB 主机地址。
//   - dbName：MongoDB 数据库名称。
//   - port：MongoDB 端口。
func NewMongoDBClient(username, password, host, dbName string, port int) (*mongo.Database, error) {
	var dsn string
	// authSource=admin：官方镜像通过 MONGO_INITDB_ROOT_USERNAME 创建的用户存放在 admin 库，
	// 不指定时驱动默认用路径中的库（blog）做认证，会报 AuthenticationFailed
	dsn = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=admin", username, password, host, port, dbName)

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

	return client.Database(dbName), nil
}
