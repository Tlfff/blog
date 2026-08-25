package cron

import (
	"context"
	"log"
	"time"
)

// RankRebuilder 是热榜重建接口。
type RankRebuilder interface {
	// RebuildHotRank 重建文章热榜。
	RebuildHotRank(ctx context.Context) error
}

// RankSyncJob 定时重建文章热榜。
type RankSyncJob struct {
	rankService RankRebuilder // 文章热榜重建应用用例
}

// NewRankSyncJob 创建文章热榜校准任务。
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

// Name 返回热榜校准任务名称。
func (j *RankSyncJob) Name() string {
	return "rank_calibrate_daily"
}

// Run 执行文章热榜校准任务。
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
