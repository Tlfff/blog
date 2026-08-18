// Package redis 提供基于 Redis 的分布式锁实现。
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

// 用embed嵌入lua目录（//go:embed 必须紧邻变量声明），编译后也可正常读取脚本文件
//
//go:embed lua
var luaFS embed.FS

// RedisLock 是基于 Redis SETNX 实现的分布式锁。
type RedisLock struct {
	rdb        *redis.Client // 客户端
	key        string        // 锁Key
	value      string        // 锁唯一标识，防止误删
	expireTime time.Duration // 过期时间
	retryCount int           // 重试次数
	retryDelay time.Duration // 每次重试间隔
	locked     bool          // 是否成功持有锁
}

// 创建一把 Redis 分布式锁，key 为锁标识，expireTime 为锁自动过期时间
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

// 带重试的加锁，重试次数与间隔由锁自身配置决定
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

// 释放锁，通过 Lua 脚本保证只删除自己持有的那把锁
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

// 尝试加锁一次且不重试，value 为锁唯一标识，防止误删他人持有的锁
func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	// 1. 已经持有锁时直接返回成功
	if l.locked == true {
		return true, nil
	}
	// 2. 通过 SETNX 原子写入锁
	ok, err := l.rdb.SetNX(ctx, l.key, l.value, l.expireTime).Result()
	if err != nil {
		return false, err
	}
	// 3. 记录持有状态并返回加锁结果
	l.locked = ok
	return ok, nil
}
