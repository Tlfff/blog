package database

import (
	"blog/internal/infrastructure/config"
	"context"

	"github.com/redis/go-redis/v9"
)

// 创建 Redis 客户端并做连通性探测
func NewRedisClient(cfg config.Redis) (*redis.Client, error) {
	// 1. 按配置创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		DB:       cfg.DB,
		Password: cfg.Password,
	})
	// 2. Ping 探测连通性，错误一并返回给调用方
	_, err := rdb.Ping(context.Background()).Result()
	return rdb, err
}
