package service

import (
	"blog/internal/common"
	"blog/internal/message"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/kafka"
	"context"
	"log"
	"time"
)

type ArticleViewHistoryService struct {
	repo        *repository.ArticleViewHistoryRepository
	viewMap     *common.ViewCacheMap // 维护一个内存中的浏览历史记录，key: userID_articleID, value: lastViewTime
	kafkaClient *kafka.Client
}

// NewArticleViewHistoryService 初始化独立的浏览历史服务
func NewArticleViewHistoryService(repo *repository.ArticleViewHistoryRepository, kafkaClient *kafka.Client) *ArticleViewHistoryService {
	return &ArticleViewHistoryService{
		repo:        repo,
		viewMap:     common.NewViewCacheMap(),
		kafkaClient: kafkaClient,
	}
}

// 记录浏览历史
func (s *ArticleViewHistoryService) RecordView(ctx context.Context, userID, articleID uint64, ip string) error {
	// // 1. 防刷检查
	// if !s.viewMap.CheckAndSet(userID, articleID, ip, 10*time.Minute) {
	// 	// 10 分钟内已记录过，不发送消息
	// 	return
	// }

	// 2. 发送 Kafka 消息
	return s.sendViewHistoryToKafka(ctx, userID, articleID)
}

// ---------------------------- 发送Kafka消息 ----------------------------
// 发送浏览历史消息到 Kafka topic
func (s *ArticleViewHistoryService) sendViewHistoryToKafka(ctx context.Context, userID, articleID uint64) error {
	// 1. 检查 Kafka 客户端是否可用
	if s.kafkaClient == nil {
		return common.ErrKafkaClientClosed
	}
	producer := s.kafkaClient.GetProducer()
	if producer == nil {
		return common.ErrKafkaClientClosed
	}

	// 2. 构造消息
	msg := &message.ViewHistoryMsg{
		ArticleID:   articleID,
		UserID:      userID,
		CreatedTime: time.Now(),
	}

	// 3. 使用传入的 ctx，创建子sendCtx上下文，并设置发送超时时间，确保 Kafka 发送在 5s 内完成
	// 如果 HTTP 请求超时或被取消，这里的 Kafka 发送也会立即终止
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := producer.SendViewHistory(sendCtx, msg); err != nil {
		log.Printf("[Kafka] 发送浏览历史失败, user: %d, article: %d, err: %v", userID, articleID, err)
		return err
	}

	log.Printf("[Kafka] 发送浏览历史成功, user: %d, article: %d", userID, articleID)
	return nil
}

// ---------------------------- 消费Kafka消息 ----------------------------

// 处理从 notification topic 消费的消息，记录浏览历史
func (s *ArticleViewHistoryService) CreateViewHistory(ctx context.Context, userID, articleID uint64, timestamp time.Time) error {
	// 1. 如果是登录用户，记录浏览历史
	if userID > 0 {
		history := &model.ArticleViewHistory{
			UserID:      userID,
			ArticleID:   articleID,
			CreatedTime: timestamp,
			UpdatedTime: timestamp,
		}
		if err := s.repo.CreateViewHistory(ctx, history); err != nil {
			log.Printf("写入浏览历史失败 uid=%d aid=%d err=%v", userID, articleID, err)
		}
	}

	// 2. 文章主表的 view_count 原子自增 1
	if err := s.repo.IncrementViewCount(ctx, articleID); err != nil {
		log.Printf("阅读量自增失败 aid=%d err=%v", articleID, err)
		return err
	}
	return nil
}
