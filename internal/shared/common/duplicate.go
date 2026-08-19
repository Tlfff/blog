package common

import (
	"sync"
	"time"
)

// DuplicateChecker 是基于内存的重复提交检查器。
type DuplicateChecker struct {
	mu    sync.RWMutex         // 因为有后台协程写，核心地方需要换成 Lock
	cache map[string]time.Time // key 到最近一次提交时间的映射
}

// Duplicate 是全局共享的重复提交检查器实例
var Duplicate = NewDuplicateChecker()

// 防止内存积压
// 创建重复提交检查器，并启动后台清理协程防止内存积压
func NewDuplicateChecker() *DuplicateChecker {
	// 1. 初始化缓存字典
	d := &DuplicateChecker{
		cache: make(map[string]time.Time),
	}
	// 2. 启动后台清理任务，每 5 分钟扫一次内存
	// 启动后台清理任务，每 5 分钟扫一次内存
	go d.startGC(5 * time.Minute)
	// 3. 返回可用实例
	return d
}

// 检查是否重复提交
// 检查指定 key 在 expire 时间内是否已提交过
// 返回 true 表示属于重复提交
func (d *DuplicateChecker) Check(key string, expire time.Duration) bool {
	now := time.Now()

	// 1. 加写锁，保证检查与写入的原子性
	d.mu.Lock() // 修改数据，用写锁
	defer d.mu.Unlock()

	// 2. 命中缓存且未超过冷却时间，判定为重复提交
	if lastTime, ok := d.cache[key]; ok {
		if now.Sub(lastTime) < expire {
			return true // 重复提交
		}
	}

	// 3. 未重复则记录本次提交时间
	d.cache[key] = now
	return false
}

// 定时清理过期 key，防止内存泄漏
func (d *DuplicateChecker) startGC(interval time.Duration) {
	// 创建一个每 5 （interval）分钟滴答一次的定时器
	ticker := time.NewTicker(interval)
	// 配合 for range 循环，协程会在这里每 5 分钟被唤醒一次
	// 1. 每次定时器触发后加写锁遍历缓存
	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for k, t := range d.cache {
			// 2. 超过保留时间（2秒）的记录直接删除
			// 如果超出了一定范围（2秒），执行删除
			if now.Sub(t) > 2*time.Second {
				delete(d.cache, k)
			}
		}
		d.mu.Unlock() //立即释放
	}
}
