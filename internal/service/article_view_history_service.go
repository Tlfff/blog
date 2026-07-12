package service

import (
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"log"
	"time"
)

type ArticleViewHistoryService struct {
	repo    *repository.ArticleViewHistoryRepository
	viewMap *common.ViewCacheMap // 维护一个内存中的浏览历史记录，key: userID_articleID, value: lastViewTime
}

// NewArticleViewHistoryService 初始化独立的浏览历史服务
func NewArticleViewHistoryService(repo *repository.ArticleViewHistoryRepository) *ArticleViewHistoryService {
	return &ArticleViewHistoryService{
		repo:    repo,
		viewMap: common.NewViewCacheMap(),
	}
}

// RecordView 核心业务逻辑：异步检查并记录浏览历史
func (s *ArticleViewHistoryService) RecordView(userID, articleID uint64, ip string) {
	// 开启异步协程，让调用方瞬间返回，不阻塞主协程
	go func() {
		// 捕获异常
		defer func() {
			if err := recover(); err != nil {
				log.Printf("协程异常，方法：%s,异常：%v", "recordView", err)
			}
		}()

		// 设置个过期时间3s
		newCtx, cancle := context.WithTimeout(context.Background(), 3*time.Second)
		// 程序退出前释放上下文资源
		defer cancle()
		//  如果返回 true，说明该用户在这 10 分钟内是第一次看这篇文章
		if s.viewMap.CheckAndSet(userID, articleID, ip, 10*time.Minute) {
			// 如果是登录用户，记录浏览历史
			if userID > 0 {
				// 1. 构造流水记录
				history := &model.ArticleViewHistory{
					UserID:    userID,
					ArticleID: articleID,
				}

				// 2. 写入浏览历史表
				if err := s.repo.CreateViewHistory(newCtx, history); err != nil {
					log.Printf("写入浏览历史失败 uid=%d aid=%d err=%v", userID, articleID, err)
				}
			}

			// 3. 文章主表的 view_count 原子自增 1
			if err := s.repo.IncrementViewCount(newCtx, articleID); err != nil {
				log.Printf("阅读量自增失败 aid=%d err=%v", articleID, err)
			}
		}
	}()
}
