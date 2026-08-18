package community

import (
	"blog/internal/consts"
	domaincommunity "blog/internal/domain/community"
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type hotRankStoreAdapter struct {
	rdb *redis.Client
}

// NewHotRankStore 提供热榜 Redis ZSet 读写。
func NewHotRankStore(rdb *redis.Client) domaincommunity.HotRankStore {
	return &hotRankStoreAdapter{rdb: rdb}
}

func (a *hotRankStoreAdapter) GetTop(ctx context.Context, limit int) ([]domaincommunity.HotRankItem, error) {
	values, err := a.rdb.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:   consts.KeyArticleHotRankZSet,
		Start: 0,
		Stop:  int64(limit - 1),
		Rev:   true,
	}).Result()
	if err != nil {
		return nil, err
	}
	items := make([]domaincommunity.HotRankItem, 0, len(values))
	for _, value := range values {
		member, ok := value.Member.(string)
		if !ok {
			continue
		}
		id, err := strconv.ParseUint(member, 10, 64)
		if err != nil {
			continue
		}
		items = append(items, domaincommunity.HotRankItem{
			ArticleID: id,
			Hot:       value.Score,
		})
	}
	return items, nil
}

func (a *hotRankStoreAdapter) Rebuild(ctx context.Context, entries []domaincommunity.HotRankItem) error {
	key := consts.KeyArticleHotRankZSet
	if len(entries) == 0 {
		return a.rdb.Del(ctx, key).Err()
	}
	zs := make([]redis.Z, 0, len(entries))
	for _, entry := range entries {
		zs = append(zs, redis.Z{
			Score:  entry.Hot,
			Member: strconv.FormatUint(entry.ArticleID, 10),
		})
	}
	pipe := a.rdb.TxPipeline()
	pipe.Del(ctx, key)
	pipe.ZAdd(ctx, key, zs...)
	_, err := pipe.Exec(ctx)
	return err
}
