package domain

import "time"

// LikeCreated 表示当前基线中的文章点赞通知事件。
type LikeCreated struct {
	UserID     uint64     // 点赞用户唯一标识
	Target     LikeTarget // 点赞目标类型
	TargetID   uint64     // 点赞目标唯一标识
	OccurredAt time.Time  // 点赞发生时间
}
