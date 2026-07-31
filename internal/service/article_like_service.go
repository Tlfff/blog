package service

import (
	"blog/internal/consts"
	"blog/internal/message"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/database"
	"blog/pkg/kafka"
	redisUtil "blog/pkg/util/redis"
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ArticleLikeService struct {
	artlikeRepo repository.ArticleLikeRepository
	artRepo     *repository.ArticleRepository
	rdb         *redis.Client
	ntfService  *NotificationService
	userRepo    *repository.UserRepository
	kafkaClient *kafka.Client
}

func NewArticleLikeService(artLikeRepo repository.ArticleLikeRepository, artRepo *repository.ArticleRepository, rbd *redis.Client, ntfService *NotificationService, userRepo *repository.UserRepository, kafkaClient *kafka.Client) *ArticleLikeService {
	return &ArticleLikeService{
		artlikeRepo: artLikeRepo,
		artRepo:     artRepo,
		rdb:         rbd,
		ntfService:  ntfService,
		userRepo:    userRepo,
		kafkaClient: kafkaClient,
	}
}

// 点赞文章，返回nil则为点赞成功
func (s *ArticleLikeService) ArticleLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 判断用户是否点赞
	like, err := s.checkIsLike(ctx, articleID, userID)
	if err != nil {
		return err
	}
	if like {
		return nil
	}
	// 2. 更新数据库
	// 查找之前是否有该用户记录
	ok, err := s.artlikeRepo.FindRecord(ctx, userID, articleID)
	if err != nil {
		return err
	}
	// 2.1 事务内：写点赞记录 + 更新点赞数
	err = database.RunTx(ctx, s.artlikeRepo.GetDB(), func(tx *gorm.DB) error {
		if ok {
			if err := s.artlikeRepo.Update(ctx, tx, userID, articleID, model.ArticleLiked); err != nil {
				return err
			}
		} else {
			newLike := &model.ArticleLike{
				UserID:    userID,
				ArticleID: articleID,
				Status:    model.ArticleLiked,
			}
			if err := s.artlikeRepo.Insert(ctx, tx, newLike); err != nil {
				return err
			}
		}
		return s.artlikeRepo.UpdateArticleLikeCountDelta(ctx, tx, articleID, 1)
	})
	if err != nil {
		return err
	}
	// 3. 更新缓存
	key := s.getArticleLikeKey(articleID)
	if err := s.rdb.SAdd(ctx, key, userID).Err(); err != nil {
		log.Printf("更新点赞缓存失败,article_id:%d,user_id:%d,err:%v", articleID, userID, err)
	}
	// // 4. 更新排行榜
	// if err := s.updateRankZSet(ctx, articleID); err != nil {
	// 	log.Printf("更新排行榜失败,article_id:%d,err:%v", articleID, err)
	// }

	// 5. 异步发送通知消息到Kafka
	s.sendLikeNotificationToKafka(userID, articleID)
	return nil
}

// 取消点赞文章，返回nil则为取消点赞成功
func (s *ArticleLikeService) ArticleCancelLike(ctx context.Context, userID, articleID uint64) error {
	// 1. 判断用户是否点赞
	liked, err := s.checkIsLike(ctx, articleID, userID)
	if err != nil {
		return err
	}
	if !liked {
		return nil
	}

	// 2. 更新数据库
	// 查找之前是否有该用户记录
	ok, err := s.artlikeRepo.FindRecord(ctx, userID, articleID)
	if err != nil {
		return err
	}
	// 2.1 事务内：写点赞记录 + 更新点赞数
	err = database.RunTx(ctx, s.artlikeRepo.GetDB(), func(tx *gorm.DB) error {
		if ok {
			if err := s.artlikeRepo.Update(ctx, tx, userID, articleID, model.ArticleCancelLiked); err != nil {
				return err
			}
		} else {
			newLike := &model.ArticleLike{
				UserID:    userID,
				ArticleID: articleID,
				Status:    model.ArticleCancelLiked,
			}
			if err := s.artlikeRepo.Insert(ctx, tx, newLike); err != nil {
				return err
			}
		}
		return s.artlikeRepo.UpdateArticleLikeCountDelta(ctx, tx, articleID, -1)
	})
	if err != nil {
		return err
	}

	// 3. 更新缓存
	key := s.getArticleLikeKey(articleID)
	if err := s.rdb.SRem(ctx, key, userID).Err(); err != nil {
		log.Printf("更新取消点赞缓存失败,article_id:%d,user_id:%d,err:%v", articleID, userID, err)
	}

	// // 4. 更新排行榜
	// if err := s.updateRankZSet(ctx, articleID); err != nil {
	// 	log.Printf("更新排行榜失败,article_id:%d,err:%v", articleID, err)
	// }

	return nil
}

// 查找set是否存在
func (s *ArticleLikeService) checkSetIsExist(ctx context.Context, key string) (bool, error) {
	exist, err := s.rdb.Exists(ctx, key).Result()
	return exist > 0, err
}

// 判断用户是否点赞
func (s *ArticleLikeService) checkIsLike(ctx context.Context, articleID, userID uint64) (bool, error) {
	// 1. set是否存在
	key := s.getArticleLikeKey(articleID)
	ok, err := s.checkSetIsExist(ctx, key)
	if err != nil {
		return false, err
	}
	// 2. 存在，直接查找用户是否点赞
	if ok {
		return s.rdb.SIsMember(ctx, key, userID).Result()
	}
	// 3. set不存在，尝试获取互斥锁
	lockKey := s.getLockArticleLikeKey(articleID)
	r := redisUtil.NewRedisLock(s.rdb, lockKey, consts.LockExpirePeriod)
	lock, err := r.TryLock(ctx)
	// 如果枷锁时出现问题，降级直接查库
	if err != nil {
		return s.artlikeRepo.IsLiked(ctx, userID, articleID)
	}
	// 如果加锁成功，则拉取缓存，同时判断用户是否点赞
	if lock {
		defer r.UnLock(ctx)
		return s.rebuildSetAndCheck(ctx, articleID, userID)
	}
	return s.artlikeRepo.IsLiked(ctx, userID, articleID)
}

// 重建缓存，拉取数据库数据到set
func (s *ArticleLikeService) rebuildSetAndCheck(ctx context.Context, articleID, userID uint64) (bool, error) {
	// 1. 获取所有点赞用户的id
	userIDs, err := s.artlikeRepo.GetLikedUserIDs(ctx, articleID)
	if err != nil {
		return false, err
	}
	// 2. 查询用户是否点赞，同时将结果转换成sadd需要的类型
	isLike := false
	members := make([]interface{}, 0, len(userIDs))
	for _, id := range userIDs {
		members = append(members, id)
		if id == userID {
			isLike = true
		}
	}

	// 3.批量重建set缓存
	key := s.getArticleLikeKey(articleID)
	// 如果没有用户存在，缓存一个占位符，防止缓存穿透，不可能存在0用户
	if len(members) == 0 {
		members = append(members, 0)
	}
	if err := s.rdb.SAdd(ctx, key, members...).Err(); err != nil {
		log.Printf("重建文章点赞set失败,article_id:%d,%s", articleID, err)
	} else {
		s.rdb.Expire(ctx, key, consts.ExpirePeriod)
	}
	return isLike, nil
}

// 更新zset排行榜
func (s *ArticleLikeService) updateRankZSet(ctx context.Context, articleID uint64) error {
	// 1. 查询该文章最新的浏览量、点赞数、评论数
	res, err := s.artRepo.GetHot(ctx, articleID)
	if err != nil {
		return err
	}

	// 2. 计算总热度
	totalHeat := calcHotScore(res.ViewCount, res.LikeCount, res.CommentCount)

	// 3. 覆盖写入zset：member已存在则更新分数，不存在则新增
	if err := s.rdb.ZAdd(ctx, consts.KeyArticleHotRankZSet, redis.Z{
		Score:  float64(totalHeat),
		Member: articleID,
	}).Err(); err != nil {
		return err
	}

	// 4. 裁剪回前100，超出的从分数最低的开始删除
	if err := s.rdb.ZRemRangeByRank(ctx, consts.KeyArticleHotRankZSet, 0, -101).Err(); err != nil {
		return err
	}

	return nil
}

// 计算文章热度
func calcHotScore(viewCount, likeCount, commentCount uint32) float64 {
	return float64(1*viewCount + 2*likeCount + 1*commentCount)
}

// 判断用户是否点赞过该文章，供文章详情页展示使用
func (s *ArticleLikeService) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	return s.checkIsLike(ctx, articleID, userID)
}

// ------------------------------ key 拼接辅助函数 ------------------------------
func (s *ArticleLikeService) getArticleLikeKey(articleID uint64) string {
	return consts.KeyLikeArticlePre + strconv.FormatUint(articleID, 10)
}
func (s *ArticleLikeService) getLockArticleLikeKey(articleID uint64) string {
	return consts.KeyLockLikeArticle + strconv.FormatUint(articleID, 10)
}

// ------------------------------ 发送通知函数 ------------------------------
func (s *ArticleLikeService) asyncSendLikeNotification(userID, articleID uint64) {
	go func() {
		// 1. 捕获异常
		defer func() {
			if err := recover(); err != nil {
				log.Printf("协程异常，方法：%s,异常：%v", "recordView", err)
			}
		}()
		// 2. 创建新上下文，设置个过期时间3s
		newCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		// 3. 获取文章内容
		article, err := s.artRepo.FindArticleByID(newCtx, articleID)
		if err != nil {
			log.Printf("[Notification] 获取文章失败, article_id:%d, err:%v", articleID, err)
			return
		}
		// 4. 获取用户信息
		user, err := s.userRepo.FindUserByID(newCtx, userID)
		// 5. 发送通知
		err = s.ntfService.SendLikeArticleNotification(newCtx, user.ID, user.Nickname, user.Avatar, article.AuthorID, article.ID, article.Title)
		if err != nil {
			log.Printf("[Notification] 发送点赞通知失败, article_id:%d, user_id:%d, err:%v", articleID, userID, err)
		}

	}()
}

// 通过 Kafka 异步发送点赞通知
func (s *ArticleLikeService) sendLikeNotificationToKafka(userID, articleID uint64) {
	// 1. 检查 Kafka 客户端是否可用
	if s.kafkaClient == nil {
		return
	}
	producer := s.kafkaClient.GetProducer()
	if producer == nil {
		return
	}

	// 2. 构造消息
	msg := &message.NotificationMsg{
		NotifyType:  model.NotifyTypeLikeArticle,
		SenderID:    userID,
		TargetID:    articleID,
		CreatedTime: time.Now(),
	}

	// 3. 异步发送，ACK 确认在后台 goroutine 中等待，不阻塞点赞主流程
	producer.SendNotificationAsync(msg)
}

// ------------------------------ 消费通知消息函数 ------------------------------
// 创建点赞通知（供 Kafka 消费者调用）
func (s *ArticleLikeService) CreateLikeNotification(ctx context.Context, msg *message.NotificationMsg) error {
	log.Printf("发送点赞文章通知,article_id:%d,user_id:%d", msg.TargetID, msg.SenderID)
	// 1. 根据 TargetID 查询文章
	article, err := s.artRepo.FindArticleByID(ctx, msg.TargetID)
	if err != nil {
		log.Printf("[MQ] 获取文章信息失败, article_id: %d, err: %v", msg.TargetID, err)
		return err
	}

	// 2. 根据 SenderID 查询用户
	user, err := s.userRepo.FindUserByID(ctx, msg.SenderID)
	if err != nil {
		log.Printf("[MQ] 获取用户信息失败, user_id: %d, err: %v", msg.SenderID, err)
		return err
	}

	// 3. 发送通知
	err = s.ntfService.SendLikeArticleNotification(
		ctx,
		msg.SenderID,
		user.Nickname,
		user.Avatar,
		article.AuthorID,
		msg.TargetID,
		article.Title,
	)
	if err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}

	return nil
}
