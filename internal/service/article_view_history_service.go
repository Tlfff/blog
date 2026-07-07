package service

import (
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
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
		// 捕获panic，防止服务崩溃
		defer func() {
			if err := recover(); err != nil {
				// 打印崩溃日志，接入日志框架
				log.Printf("浏览统计协程panic recover: %v", err)
			}
		}()
		// todo: 传入context
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
				_ = s.repo.CreateViewHistory(history)
			}

			// 3. 文章主表的 view_count 原子自增 1
			_ = s.repo.IncrementViewCount(articleID)
		}
	}()
}
