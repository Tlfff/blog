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

// likeCacheAdapter 是点赞状态缓存适配器，缓存缺失时回源数据库并重建。
type likeCacheAdapter struct {
	rdb         *redis.Client                         // Redis 客户端
	articleRepo domaincommunity.ArticleLikeRepository // 文章点赞仓储，用于回源重建缓存
	commentRepo domaincommunity.CommentLikeRepository // 评论点赞仓储，用于回源重建缓存
}

// NewLikeCache 提供文章/评论点赞状态缓存与冷启动重建。
func NewLikeCache(
	rdb *redis.Client,
	articleRepo domaincommunity.ArticleLikeRepository,
	commentRepo domaincommunity.CommentLikeRepository,
) domaincommunity.LikeCache {
	return &likeCacheAdapter{rdb: rdb, articleRepo: articleRepo, commentRepo: commentRepo}
}

// 查询用户是否点赞过目标对象，缓存未命中时加锁重建缓存
func (a *likeCacheAdapter) IsLiked(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) (bool, error) {
	// 1. 缓存命中则直接用 set 判断成员是否存在
	key := likeKey(target, targetID)
	exists, err := a.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if exists > 0 {
		return a.rdb.SIsMember(ctx, key, userID).Result()
	}

	// 2. 缓存未命中，尝试加分布式锁，避免并发重建打穿数据库
	lockKey := likeLockKey(target, targetID)
	lock := redisUtil.NewRedisLock(a.rdb, lockKey, consts.LockExpirePeriod)
	acquired, err := lock.TryLock(ctx)
	// 2.1 加锁异常时直接回源数据库
	if err != nil {
		return a.isLikedFromDB(ctx, target, targetID, userID)
	}
	// 2.2 抢到锁的请求负责重建缓存
	if acquired {
		defer lock.UnLock(ctx)
		return a.rebuildAndCheck(ctx, target, targetID, userID)
	}
	// 2.3 未抢到锁的请求直接回源数据库
	return a.isLikedFromDB(ctx, target, targetID, userID)
}

// 把用户加入目标对象的点赞缓存集合
func (a *likeCacheAdapter) Add(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) error {
	if err := a.rdb.SAdd(ctx, likeKey(target, targetID), userID).Err(); err != nil {
		log.Printf("更新点赞缓存失败,target:%s,target_id:%d,user_id:%d,err:%v", target, targetID, userID, err)
		return err
	}
	return nil
}

// 把用户从目标对象的点赞缓存集合中移除
func (a *likeCacheAdapter) Remove(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) error {
	if err := a.rdb.SRem(ctx, likeKey(target, targetID), userID).Err(); err != nil {
		log.Printf("更新取消点赞缓存失败,target:%s,target_id:%d,user_id:%d,err:%v", target, targetID, userID, err)
		return err
	}
	return nil
}

// 从数据库重建点赞缓存集合，并顺带返回当前用户的点赞状态
func (a *likeCacheAdapter) rebuildAndCheck(ctx context.Context, target domaincommunity.LikeTarget, targetID, userID uint64) (bool, error) {
	// 1. 按目标类型从数据库查询全部点赞用户ID
	var userIDs []uint64
	var err error
	switch target {
	case domaincommunity.LikeTargetArticle:
		userIDs, err = a.articleRepo.GetLikedUserIDs(ctx, targetID)
	case domaincommunity.LikeTargetComment:
		userIDs, err = a.commentRepo.GetLikedUserIDs(ctx, targetID)
	}
	// 2. 查询失败直接返回错误
	if err != nil {
		return false, err
	}
	// 3. 遍历点赞用户，组装 set 成员并判断当前用户是否点赞
	isLiked := false
	members := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		members = append(members, id)
		if id == userID {
			isLiked = true
		}
	}
	// 4. 空集合写入哨兵成员 0，避免缓存穿透
	key := likeKey(target, targetID)
	if len(members) == 0 {
		members = append(members, 0)
	}
	// 5. 写入缓存并设置过期时间，写失败只记日志不影响返回结果
	if err := a.rdb.SAdd(ctx, key, members...).Err(); err != nil {
		log.Printf("重建点赞 set 失败,target:%s,target_id:%d,err:%v", target, targetID, err)
	} else {
		a.rdb.Expire(ctx, key, consts.ExpirePeriod)
	}
	return isLiked, nil
}

// 直接查询数据库判断点赞状态，作为缓存不可用时的兜底
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

// likeCountStoreAdapter 是评论点赞数缓存读取适配器。
type likeCountStoreAdapter struct {
	rdb *redis.Client // Redis 客户端
}

// NewLikeCountStore 读取评论点赞数 Redis 缓存，缺失时由调用方使用评论表计数兜底。
func NewLikeCountStore(rdb *redis.Client) domaincommunity.LikeCountStore {
	return &likeCountStoreAdapter{rdb: rdb}
}

// 批量读取评论点赞数缓存，缓存缺失的评论不会出现在返回结果中
func (a *likeCountStoreAdapter) GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error) {
	// 1. 入参为空时直接返回空结果
	result := make(map[uint64]uint64)
	if len(commentIDs) == 0 {
		return result, nil
	}
	// 2. 拼装各评论点赞数的缓存 Key
	keys := make([]string, 0, len(commentIDs))
	for _, id := range commentIDs {
		keys = append(keys, consts.KeyCommentLikeCountPrefix+strconv.FormatUint(id, 10))
	}
	// 3. 批量读取缓存
	vals, err := a.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return result, err
	}
	// 4. 解析命中的计数值，非法或缺失的直接跳过
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

// 拼装点赞集合的缓存 Key，按目标类型选择前缀
func likeKey(target domaincommunity.LikeTarget, targetID uint64) string {
	prefix := consts.KeyLikeArticlePre
	if target == domaincommunity.LikeTargetComment {
		prefix = consts.KeyLikeCommentPre
	}
	return prefix + strconv.FormatUint(targetID, 10)
}

// 拼装点赞缓存重建用的分布式锁 Key，按目标类型选择前缀
func likeLockKey(target domaincommunity.LikeTarget, targetID uint64) string {
	prefix := consts.KeyLockLikeArticle
	if target == domaincommunity.LikeTargetComment {
		prefix = consts.KeyLockLikeComment
	}
	return prefix + strconv.FormatUint(targetID, 10)
}
