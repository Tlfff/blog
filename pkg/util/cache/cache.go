package cache

import (
	redisUtil "blog/pkg/util/redis"
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// 拉取数据到缓存，双重检查的互斥锁保证效率
// ctx: 上下文
// rdb: redis客户端
// cacheKey: 目标缓存key
// lockKey: 分布式锁key
// lockValue: 锁唯一标识
// lockExpire: 锁过期时长
// retryCount: 加锁重试次数
// retryDelay: 加锁重试间隔
// checkExists: 检查缓存是否存在的函数 func(ctx) (exists bool, err error)
// initCache: 缓存不存在时执行初始化函数 func(ctx) error
func DoubleCheckInitCache(
	ctx context.Context,
	rdb *redis.Client,
	cacheKey string,
	lockKey string,
	lockValue string,
	lockExpire time.Duration,
	retryCount int,
	retryDelay time.Duration,
	checkExists func(ctx context.Context) (bool, error),
	initCache func(ctx context.Context) error,
) error {
	// 1. 第一次检查缓存是否存在
	exists, err := checkExists(ctx)
	// 连接、网络等问题
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// 2. 获取分布式锁
	lock := redisUtil.NewRedisLock(rdb, lockKey, lockExpire, retryCount, retryDelay)
	err = lock.Lock(ctx)
	if err != nil {
		return err
	}
	defer func() { lock.UnLock(ctx) }()

	// 3. 第二次检查缓存是否存在（双重检查）
	exists, err = checkExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// 4. 执行缓存初始化逻辑
	err = initCache(ctx)
	if err != nil {
		return err
	}
	return nil
}
