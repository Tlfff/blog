package consts

import "time"

const (
	LockExpire       = 3 * time.Second       // 锁过期时间
	RetryCount       = 5                     // 重试次数
	RetryDelay       = 50 * time.Millisecond // 重试间隔
	ExpirePeriod     = 7 * 24 * time.Hour    // key过期时间，7天
	LockExpirePeriod = 3 * time.Second       // 锁过期时间，3s
)
