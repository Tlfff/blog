package cron

import (
	"blog/internal/service"
	"context"
	"log"
	"time"
)

type LikeSyncJob struct {
	likeService *service.LikeService
}

func NewLikeSyncJob(likeService *service.LikeService) *LikeSyncJob {
	return &LikeSyncJob{
		likeService: likeService,
	}
}

// Spec 每天 02:00:00 执行（秒格式）
// return "0 0 2 * * *"
func (j *LikeSyncJob) Spec() string {
	return "0 * * * * *"
}

func (j *LikeSyncJob) Name() string {
	return "like_sync_daily"
}

// 1. 超时控制：由于是大批量的 Scan 和 DB 批处理，本方法内部强制包裹了 30 分钟的硬性超时防线。
// 2. 顺序刷盘：先同步文章点赞，再同步评论点赞。
func (j *LikeSyncJob) Run(ctx context.Context) error {
	log.Printf("[Cron][%s] 开始执行每日点赞刷盘任务", j.Name())

	// 1. 注入最大允许生命周期，防止在大规模扫描或数据库死锁时导致协程无期限阻塞
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	// 2. 执行点赞刷盘
	if err := j.likeService.SyncAllLikes(ctx); err != nil {
		return err
	}

	return nil
}
