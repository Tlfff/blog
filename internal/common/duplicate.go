package common

import (
	"sync"
	"time"
)

type DuplicateChecker struct {
	mu    sync.RWMutex // 因为有后台协程写，核心地方需要换成 Lock
	cache map[string]time.Time
}

var Duplicate = NewDuplicateChecker()

// 防止内存积压
func NewDuplicateChecker() *DuplicateChecker {
	d := &DuplicateChecker{
		cache: make(map[string]time.Time),
	}
	// 启动后台清理任务，每 5 分钟扫一次内存
	go d.startGC(5 * time.Minute)
	return d
}

// 检查是否重复提交
func (d *DuplicateChecker) Check(key string, expire time.Duration) bool {
	now := time.Now()

	d.mu.Lock() // 修改数据，用写锁
	defer d.mu.Unlock()

	if lastTime, ok := d.cache[key]; ok {
		if now.Sub(lastTime) < expire {
			return true // 重复提交
		} else {
			delete(d.cache, key) // 惰性删除
		}
	}

	d.cache[key] = now
	return false
}

// 定时清理过期 key，防止内存泄漏
func (d *DuplicateChecker) startGC(interval time.Duration) {
	// 创建一个每 5 （interval）分钟滴答一次的定时器
	ticker := time.NewTicker(interval)
	// 配合 for range 循环，协程会在这里每 5 分钟被唤醒一次
	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for k, t := range d.cache {
			// 如果超出了一定范围（2秒），执行删除
			if now.Sub(t) > 2*time.Second {
				delete(d.cache, k)
			}
		}
		d.mu.Unlock() //立即释放
	}
}
