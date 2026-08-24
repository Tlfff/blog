package infrastructure

import (
	commentdomain "blog/internal/comment/domain"
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const commentLikeCountPrefix = "like:comment:count:" // 评论点赞数兼容 Redis Key 前缀

type likeCountQuery struct {
	rdb *redis.Client // Redis 客户端
}

// NewLikeCountQuery 创建 Comment 拥有的点赞数只读查询 Adapter。
func NewLikeCountQuery(rdb *redis.Client) commentdomain.LikeCountQuery {
	return &likeCountQuery{rdb: rdb}
}

// GetCommentLikeCounts 批量查询评论点赞数缓存。
func (q *likeCountQuery) GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error) {
	result := make(map[uint64]uint64)
	if len(commentIDs) == 0 {
		return result, nil
	}
	keys := make([]string, 0, len(commentIDs))
	for _, id := range commentIDs {
		keys = append(keys, commentLikeCountPrefix+strconv.FormatUint(id, 10))
	}
	values, err := q.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return result, err
	}
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			continue
		}
		count, err := strconv.ParseUint(text, 10, 64)
		if err == nil {
			result[commentIDs[index]] = count
		}
	}
	return result, nil
}
