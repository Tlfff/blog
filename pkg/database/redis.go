package database

import (
	"blog/config"
	"context"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg config.Redis) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		DB:       cfg.DB,
		Password: cfg.Password,
	})
	// 连通测试
	_, err := rdb.Ping(context.Background()).Result()
	return rdb, err
}
