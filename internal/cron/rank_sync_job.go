package cron

import (
	"blog/internal/service"
	"context"
	"log"
	"time"
)

type RankSyncJob struct {
	rankService *service.ArticleRankService
}

func NewRankSyncJob(rankService *service.ArticleRankService) *RankSyncJob {
	return &RankSyncJob{
		rankService: rankService,
	}
}

// Spec 每天 02:10:00 执行，和点赞任务错开时间
// return "0 10 2 * * *"
func (j *RankSyncJob) Spec() string {
	return "0 * * * * *"
}

func (j *RankSyncJob) Name() string {
	return "rank_calibrate_daily"
}

func (j *RankSyncJob) Run(ctx context.Context) error {
	log.Printf("[Cron][%s] 开始执行每日榜单校准任务", j.Name())
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	if err := j.rankService.DailyCalibrate(ctx); err != nil {
		log.Printf("[Cron][%s] 榜单校准失败: %v", j.Name(), err)
		return err
	}
	log.Printf("[Cron][%s] 每日榜单校准完成", j.Name())
	return nil
}
