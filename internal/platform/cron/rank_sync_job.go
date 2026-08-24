package cron

import (
	"context"
	"log"
	"time"
)

// RankRebuilder 是热榜重建接口。
type RankRebuilder interface {
	RebuildHotRank(ctx context.Context) error
}

type RankSyncJob struct {
	rankService RankRebuilder
}

func NewRankSyncJob(rankService RankRebuilder) *RankSyncJob {
	return &RankSyncJob{
		rankService: rankService,
	}
}

// Spec 每小时 执行
// return "0 0 * * * *"
func (j *RankSyncJob) Spec() string {
	return "0 0 * * * *"
}

func (j *RankSyncJob) Name() string {
	return "rank_calibrate_daily"
}

func (j *RankSyncJob) Run(ctx context.Context) error {
	log.Printf("[Cron][%s] 开始执行每日榜单校准任务", j.Name())
	// 设置一个5分钟的过期时间
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := j.rankService.RebuildHotRank(ctx); err != nil {
		log.Printf("[Cron][%s] 榜单校准失败: %v", j.Name(), err)
		return err
	}
	log.Printf("[Cron][%s] 每日榜单校准完成", j.Name())
	return nil
}
