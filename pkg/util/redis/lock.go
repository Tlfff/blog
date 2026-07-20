package redis

import (
	"blog/internal/common"
	"context"
	"embed"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// 用embed嵌入整个lua目录，编译后也可正常读取脚本文件
var luaFS embed.FS

type RedisLock struct {
	rdb        *redis.Client // 客户端
	key        string        // 锁Key
	value      string        // 锁唯一标识，防止误删
	expireTime time.Duration // 过期时间
	retryCount int           // 重试次数
	retryDelay time.Duration // 每次重试间隔
	locked     bool          // 是否成功持有锁
}

func NewRedisLock(rdb *redis.Client, key string, expireTime time.Duration) *RedisLock {
	return &RedisLock{
		rdb:        rdb,
		key:        key,
		value:      uuid.NewString(),
		expireTime: expireTime,
	}
}

// 全局缓存解锁脚本，避免每次读取文件
var unlockScript string

// 初始化加载lua脚本（包初始化时执行一次）
func init() {
	// 读取 unlock.lua 脚本
	data, err := luaFS.ReadFile("lua/unlock.lua")
	if err != nil {
		log.Println("读取redislua脚本失败: " + err.Error())
	}
	unlockScript = string(data)
}

func (l *RedisLock) RetryLock(ctx context.Context) error {
	for i := 0; i < l.retryCount; i++ {
		// 检查上下文是否过期
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// 1. 尝试获取锁，底层是SET + NX + EX 的原子写法，
		ok, err := l.rdb.SetNX(ctx, l.key, l.value, l.expireTime).Result()
		if err != nil {
			return err
		}
		if ok {
			l.locked = true
			return nil
		}
		// 2. 加锁失败重试
		time.Sleep(l.retryDelay)
	}
	// 3.加锁失败
	return common.ErrLockFailed
}

func (l *RedisLock) UnLock(ctx context.Context) error {
	// 1. 如果锁已经释放，则返回
	if !l.locked {
		return nil
	}
	// 2. 执行lua脚本并删除锁
	res, err := l.rdb.Eval(ctx, unlockScript, []string{l.key}, l.value).Result()
	if err != nil {
		return err
	}
	// 3. 当返回值为1时，说明删除锁成功
	if v, ok := res.(int64); ok && v == 1 {
		l.locked = false
		return nil
	}
	return common.ErrUnLockFailed
}

// 加锁,value为锁唯一标识，防止误删
func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	if l.locked == true {
		return true, nil
	}
	ok, err := l.rdb.SetNX(ctx, l.key, l.value, l.expireTime).Result()
	if err != nil {
		return false, err
	}
	l.locked = ok
	return ok, nil
}
