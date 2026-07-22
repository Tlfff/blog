package service

import (
	"blog/internal/consts"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/database"
	redisUtil "blog/pkg/util/redis"
	"context"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CommentLikeService struct {
	commentLikeRepo repository.CommentLikeRepository
	commentRepo     repository.CommentRepository
	rdb             *redis.Client
}

func NewCommentLikeService(commentLikeRepo repository.CommentLikeRepository, commentRepo repository.CommentRepository, rdb *redis.Client) *CommentLikeService {
	return &CommentLikeService{
		commentLikeRepo: commentLikeRepo,
		commentRepo:     commentRepo,
		rdb:             rdb,
	}
}

// 点赞评论，返回nil则为点赞成功
func (s *CommentLikeService) CommentLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 判断用户是否点赞
	liked, err := s.checkIsLike(ctx, commentID, userID)
	if err != nil {
		return err
	}
	if liked {
		return nil
	}

	// 2. 更新数据库
	ok, err := s.commentLikeRepo.FindRecord(ctx, userID, commentID)
	if err != nil {
		return err
	}
	// 事务执行：增加用户操作、修改评论点赞数
	err = database.RunTx(ctx, s.commentLikeRepo.GetDB(), func(tx *gorm.DB) error {
		if ok {
			if err := s.commentLikeRepo.Update(ctx, tx, userID, commentID, model.CommentLiked); err != nil {
				return err
			}
		} else {
			newLike := &model.CommentLike{
				UserID:    userID,
				CommentID: commentID,
				Status:    model.CommentLiked,
			}
			if err := s.commentLikeRepo.Insert(ctx, tx, newLike); err != nil {
				return err
			}
		}
		// 给对应评论的点赞数+1
		return s.commentLikeRepo.UpdateCommentLikeCountDelta(ctx, tx, commentID, 1)
	})
	if err != nil {
		return err
	}

	// 3. 更新缓存
	key := s.getCommentLikeKey(commentID)
	if err := s.rdb.SAdd(ctx, key, userID).Err(); err != nil {
		log.Printf("更新评论点赞缓存失败,comment_id:%d,user_id:%d,err:%v", commentID, userID, err)
	}

	return nil
}

// 取消点赞评论，返回nil则为取消点赞成功
func (s *CommentLikeService) CommentCancelLike(ctx context.Context, userID, commentID uint64) error {
	// 1. 判断用户是否点赞
	liked, err := s.checkIsLike(ctx, commentID, userID)
	if err != nil {
		return err
	}
	if !liked {
		return nil
	}

	// 2. 更新数据库
	// 查找之前是否有该用户记录
	ok, err := s.commentLikeRepo.FindRecord(ctx, userID, commentID)
	if err != nil {
		return err
	}
	// 事务执行：增加用户操作、修改评论点赞数
	err = database.RunTx(ctx, s.commentLikeRepo.GetDB(), func(tx *gorm.DB) error {
		// 有则更新，没有则入
		if ok {
			if err := s.commentLikeRepo.Update(ctx, tx, userID, commentID, model.CommentCancelLiked); err != nil {
				return err
			}
		} else {
			newLike := &model.CommentLike{
				UserID:    userID,
				CommentID: commentID,
				Status:    model.CommentCancelLiked,
			}
			if err := s.commentLikeRepo.Insert(ctx, tx, newLike); err != nil {
				return err
			}
		}
		// 给对应评论的点赞数-1
		return s.commentLikeRepo.UpdateCommentLikeCountDelta(ctx, tx, commentID, -1)
	})
	if err != nil {
		return err
	}

	// 3. 更新缓存
	key := s.getCommentLikeKey(commentID)
	if err := s.rdb.SRem(ctx, key, userID).Err(); err != nil {
		log.Printf("更新取消评论点赞缓存失败,comment_id:%d,user_id:%d,err:%v", commentID, userID, err)
	}

	return nil
}

// 查找set是否存在
func (s *CommentLikeService) checkSetIsExist(ctx context.Context, key string) (bool, error) {
	exist, err := s.rdb.Exists(ctx, key).Result()
	return exist > 0, err
}

// 判断用户是否点赞
func (s *CommentLikeService) checkIsLike(ctx context.Context, commentID, userID uint64) (bool, error) {
	// 1. set是否存在
	key := s.getCommentLikeKey(commentID)
	ok, err := s.checkSetIsExist(ctx, key)
	if err != nil {
		return false, err
	}
	// 2. 存在，直接查找用户是否点赞
	if ok {
		return s.rdb.SIsMember(ctx, key, userID).Result()
	}
	// 3. set不存在，尝试获取互斥锁
	lockKey := s.getLockCommentLikeKey(commentID)
	r := redisUtil.NewRedisLock(s.rdb, lockKey, consts.LockExpirePeriod)
	lock, err := r.TryLock(ctx)
	if err != nil {
		return s.commentLikeRepo.IsLiked(ctx, userID, commentID)
	}
	if lock {
		defer r.UnLock(ctx)
		return s.rebuildSetAndCheck(ctx, commentID, userID)
	}
	return s.commentLikeRepo.IsLiked(ctx, userID, commentID)
}

// 重建缓存，拉取数据库数据到set
func (s *CommentLikeService) rebuildSetAndCheck(ctx context.Context, commentID, userID uint64) (bool, error) {
	// 1. 获取所有点赞用户的id
	userIDs, err := s.commentLikeRepo.GetLikedUserIDs(ctx, commentID)
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
	key := s.getCommentLikeKey(commentID)
	// 如果没有用户存在，缓存一个占位符，防止缓存穿透，不可能存在0用户
	if len(members) == 0 {
		members = append(members, 0)
	}
	if err := s.rdb.SAdd(ctx, key, members...).Err(); err != nil {
		log.Printf("重建评论点赞set失败,comment_id:%d,%s", commentID, err)
	} else {
		s.rdb.Expire(ctx, key, consts.ExpirePeriod)
	}
	return isLike, nil
}

// ------------------------------ key 拼接辅助函数 ------------------------------
func (s *CommentLikeService) getCommentLikeKey(commentID uint64) string {
	return consts.KeyLikeCommentPre + strconv.FormatUint(commentID, 10)
}
func (s *CommentLikeService) getLockCommentLikeKey(commentID uint64) string {
	return consts.KeyLockLikeComment + strconv.FormatUint(commentID, 10)
}
