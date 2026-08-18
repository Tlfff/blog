package common

import (
	"fmt"
	"sync"
	"time"
)

// ViewCacheMap 是浏览量防刷的内存冷却缓存。
type ViewCacheMap struct {
	mu    sync.RWMutex         // 读写锁，保护 cache 并发访问
	cache map[string]time.Time // 冷却 Key 到最近一次浏览时间的映射
}

// 创建浏览冷却缓存，并启动后台清理协程
func NewViewCacheMap() *ViewCacheMap {
	v := &ViewCacheMap{
		cache: make(map[string]time.Time),
	}
	// 启动后台清理任务，每 5 分钟扫一次内存
	go v.startGC(5 * time.Minute)
	return v
}

// 检查并设置浏览冷却标记，返回 true 表示本次浏览有效应计数
func (c *ViewCacheMap) CheckAndSet(userID, articleID uint64, ip string, expire time.Duration) bool {
	// 1. 按登录态拼装冷却 Key，游客与登录用户分开统计
	var key string
	if userID == 0 {
		// 游客：用 IP + 文章ID 作为唯一的冷却 Key，更换网络ip可能会变，但是也能一定程度上防刷浏览量
		key = fmt.Sprintf("guest:%s:%d", ip, articleID)
	} else {
		// 登录用户：依然用注册的 userID
		key = fmt.Sprintf("user:%d:%d", userID, articleID)
	}
	// 1. 检查是否存在
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	// 2. 存在，则检查是否过期过期则删除
	if lastTime, ok := c.cache[key]; ok {
		if now.Sub(lastTime) < expire {
			return false // 重复浏览
		}
	}
	c.cache[key] = now
	return true
}

// 定时清理过期 key，防止内存泄漏
// 定时清理过期 key，防止内存泄漏
func (c *ViewCacheMap) startGC(interval time.Duration) {
	// 创建一个每 5分钟执行一次的定时器
	ticker := time.NewTicker(interval)
	// 配合 for range 循环，协程会在这里每 5 分钟被唤醒一次
	// 1. 每次定时器触发后加写锁遍历缓存
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, t := range c.cache {
			// 2. 超过保留时间（2秒）的记录直接删除
			// 如果超出了一定范围（2秒），执行删除
			if now.Sub(t) > 2*time.Second {
				delete(c.cache, k)
			}
		}
		c.mu.Unlock() //立即释放
	}
}
