package cron

import "context"

// CronJob 定时任务通用接口
type CronJob interface {
	Spec() string //  返回 cron 表达式

	Run(ctx context.Context) error //  执行任务逻辑

	Name() string //  任务名称（日志/监控用）
}
