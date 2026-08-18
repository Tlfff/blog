package community

import (
	"blog/internal/consts"
	domaincommunity "blog/internal/domain/community"
	redisUtil "blog/pkg/util/redis"
	"context"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type likeCacheAdapter struct {
	rdb         *redis.Client
	articleRepo domaincommunity.ArticleLikeRepository
	commentRepo domaincommunity.CommentLikeRepository
}

// NewLikeCache 提供文章/评论点赞状态缓存与冷启动重建。
func NewLikeCache(
	rdb *redis.Client,
	articleRepo domaincommunity.ArticleLikeRepository,
	commentRepo domaincommunity.CommentLikeRepository,
) domaincommunity.LikeCache {
	return &likeCacheAdapter{rdb: rdb, articleRepo: articleRepo, commentRepo: commentRepo}
}

func (a *likeCacheAdapter) IsLiked(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) (bool, error) {
	key := likeKey(target, targetID)
	exists, err := a.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists > 0 {
		return a.rdb.SIsMember(ctx, key, userID).Result()
	}

	lockKey := likeLockKey(target, targetID)
	lock := redisUtil.NewRedisLock(a.rdb, lockKey, consts.LockExpirePeriod)
	acquired, err := lock.TryLock(ctx)
	if err != nil {
		return a.isLikedFromDB(ctx, target, targetID, userID)
	}
	if acquired {
		defer lock.UnLock(ctx)
		return a.rebuildAndCheck(ctx, target, targetID, userID)
	}
	return a.isLikedFromDB(ctx, target, targetID, userID)
}

func (a *likeCacheAdapter) Add(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) error {
	if err := a.rdb.SAdd(ctx, likeKey(target, targetID), userID).Err(); err != nil {
		log.Printf("更新点赞缓存失败,target:%s,target_id:%d,user_id:%d,err:%v", target, targetID, userID, err)
		return err
	}
	return nil
}

func (a *likeCacheAdapter) Remove(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) error {
	if err := a.rdb.SRem(ctx, likeKey(target, targetID), userID).Err(); err != nil {
		log.Printf("更新取消点赞缓存失败,target:%s,target_id:%d,user_id:%d,err:%v", target, targetID, userID, err)
		return err
	}
	return nil
}

func (a *likeCacheAdapter) rebuildAndCheck(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) (bool, error) {
	var userIDs []uint64
	var err error
	switch target {
	case domaincommunity.LikeTargetArticle:
		userIDs, err = a.articleRepo.GetLikedUserIDs(ctx, targetID)
	case domaincommunity.LikeTargetComment:
		userIDs, err = a.commentRepo.GetLikedUserIDs(ctx, targetID)
	}
	if err != nil {
		return false, err
	}
	isLiked := false
	members := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		members = append(members, id)
		if id == userID {
			isLiked = true
		}
	}
	key := likeKey(target, targetID)
	if len(members) == 0 {
		members = append(members, 0)
	}
	if err := a.rdb.SAdd(ctx, key, members...).Err(); err != nil {
		log.Printf("重建点赞 set 失败,target:%s,target_id:%d,err:%v", target, targetID, err)
	} else {
		a.rdb.Expire(ctx, key, consts.ExpirePeriod)
	}
	return isLiked, nil
}

func (a *likeCacheAdapter) isLikedFromDB(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) (bool, error) {
	switch target {
	case domaincommunity.LikeTargetArticle:
		return a.articleRepo.IsLiked(ctx, userID, targetID)
	case domaincommunity.LikeTargetComment:
		return a.commentRepo.IsLiked(ctx, userID, targetID)
	default:
		return false, nil
	}
}

type likeCountStoreAdapter struct {
	rdb *redis.Client
}

// NewLikeCountStore 读取评论点赞数 Redis 缓存，缺失时由调用方使用评论表计数兜底。
func NewLikeCountStore(rdb *redis.Client) domaincommunity.LikeCountStore {
	return &likeCountStoreAdapter{rdb: rdb}
}

func (a *likeCountStoreAdapter) GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error) {
	result := make(map[uint64]uint64)
	if len(commentIDs) == 0 {
		return result, nil
	}
	keys := make([]string, 0, len(commentIDs))
	for _, id := range commentIDs {
		keys = append(keys, consts.KeyCommentLikeCountPrefix+strconv.FormatUint(id, 10))
	}
	vals, err := a.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return result, err
	}
	for i, val := range vals {
		if val != nil {
			if countStr, ok := val.(string); ok {
				if count, err := strconv.ParseUint(countStr, 10, 64); err == nil {
					result[commentIDs[i]] = count
				}
			}
		}
	}
	return result, nil
}

func likeKey(target domaincommunity.LikeTarget, targetID uint64) string {
	prefix := consts.KeyLikeArticlePre
	if target == domaincommunity.LikeTargetComment {
		prefix = consts.KeyLikeCommentPre
	}
	return prefix + strconv.FormatUint(targetID, 10)
}

func likeLockKey(target domaincommunity.LikeTarget, targetID uint64) string {
	prefix := consts.KeyLockLikeArticle
	if target == domaincommunity.LikeTargetComment {
		prefix = consts.KeyLockLikeComment
	}
	return prefix + strconv.FormatUint(targetID, 10)
}
