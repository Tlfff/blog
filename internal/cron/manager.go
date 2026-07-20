package cron

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"
)

type CronManager struct {
	cronEngine *cron.Cron
	entryIDs   map[string]cron.EntryID // 记录任务ID，便于后续管理
	jobs       []CronJob
}

func NewCronManager(jobs ...CronJob) *CronManager {
	// 全局唯一 cron 实例
	engine := cron.New(
		cron.WithSeconds(),
		// 防任务重叠执行：上次任务没跑完则跳过本次
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger), // 捕获panic防止整个cron崩溃
		),
	)

	mgr := &CronManager{
		cronEngine: engine,
		entryIDs:   make(map[string]cron.EntryID),
		jobs:       jobs,
	}

	// 循环注册所有任务
	for _, job := range jobs {
		j := job
		id, err := engine.AddFunc(j.Spec(), func() {
			err := j.Run(context.Background())
			if err != nil {
				log.Printf("[Cron] job: %s 运行错误: %v", j.Name(), err)
			}
		})
		if err != nil {
			log.Fatalf("[Cron] 注册 job: %s 失败: %v", j.Name(), err)
		}
		mgr.entryIDs[j.Name()] = id
		log.Printf("[Cron] 注册 job: %s, cron 表达式: %s", j.Name(), j.Spec())
	}

	return mgr
}

func (m *CronManager) Start() {
	m.cronEngine.Start()
	log.Println("[Cron Manager] 定时任务引擎启动")
}

func (m *CronManager) Stop() {
	ctx := m.cronEngine.Stop()
	<-ctx.Done()
	log.Println("[Cron Manager] 定时任务引擎关闭")
}
