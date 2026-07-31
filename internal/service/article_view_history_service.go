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
	kafkaClient *kafka.Client
}

// NewArticleViewHistoryService 初始化独立的浏览历史服务
func NewArticleViewHistoryService(repo *repository.ArticleViewHistoryRepository, kafkaClient *kafka.Client) *ArticleViewHistoryService {
	return &ArticleViewHistoryService{
		repo:        repo,
		kafkaClient: kafkaClient,
	}
}

// ---------------------------- 发送Kafka消息 ----------------------------
// 发送浏览历史消息到 Kafka topic
func (s *ArticleViewHistoryService) SendViewHistory(ctx context.Context, userID, articleID uint64) error {
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

	// 3， 发送消息
	if err := producer.SendViewHistory(ctx, msg); err != nil {
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
