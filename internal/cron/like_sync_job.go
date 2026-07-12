package cron

import (
	"blog/internal/service"
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type LikeSyncJob struct {
	likeService *service.LikeService
	cronEngine  *cron.Cron
}

func NewLikeSyncJob(likeService *service.LikeService) *LikeSyncJob {
	// 创建标准 crontab 规则执行引擎
	return &LikeSyncJob{
		likeService: likeService,
		cronEngine:  cron.New(),
	}
}

// Start 启动定时器组件
func (j *LikeSyncJob) Start() {
	// "0 2 * * *" 代表每天凌晨 02:00:00 自动触发
	_, err := j.cronEngine.AddFunc("0 2 * * *", func() {
		log.Println("[Cron Task] 触发每日点赞异步刷盘机制...")

		// 设定 30 分钟超时防御，防止极端情况下任务挂起
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		// 1. 同步文章点赞数据
		if err := j.likeService.SyncArticleLikesToDB(ctx); err != nil {
			log.Printf("[Cron Task Error] 点赞同步失败: %v\n", err)
		} else {
			log.Println("[Cron Task Success] 每日点赞数据已安全写回 MySQL。")
		}
		//  2. 同步评论点赞数据
		if err := j.likeService.SyncCommentLikesToDB(ctx); err != nil {
			log.Printf("[Cron Task Error] 评论点赞同步失败: %v\n", err)
		} else {
			log.Println("[Cron Task Success] 每日评论点赞数据已安全写回 MySQL。")
		}
	})

	if err != nil {
		log.Fatalf("注册点赞定时任务失败: %v", err)
	}

	j.cronEngine.Start()
	log.Println("[Cron Server] 离线 Write-Back 定时刷盘引擎启动成功...")
}

// Stop 优雅关闭（供服务关闭时释放资源）
func (j *LikeSyncJob) Stop() {
	j.cronEngine.Stop()
}
