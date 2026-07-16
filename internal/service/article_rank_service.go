package service

import (
	"blog/internal/consts"
	"blog/internal/dto/article"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/util/cache"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type ArticleRankService struct {
	articleRepo *repository.ArticleRepository
	rdb         *redis.Client
}

// 创建排行榜服务实例
func NewArticleRankService(articleRepo *repository.ArticleRepository, rdb *redis.Client) *ArticleRankService {
	return &ArticleRankService{
		articleRepo: articleRepo,
		rdb:         rdb,
	}
}

// 拼接单篇文章详情Hash的Key
func (s *ArticleRankService) getArticleInfoHashKey(articleID uint64) string {
	return fmt.Sprintf("%s%d", consts.KeyArticleInfoHashPrefix, articleID)
}

// 基础热度计算公式: HotScore = 1*浏览 + 2*点赞 + 3*评论
func (s *ArticleRankService) calcHotScore(view, like, comment uint32) float64 {
	return float64(1*view + 2*like + 3*comment)
}

// 检查排行榜ZSet是否存在
func (s *ArticleRankService) checkZSetExists(ctx context.Context) (bool, error) {
	exists, err := s.rdb.Exists(ctx, consts.KeyArticleHotRankZSet).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// 冷启动一次性全量初始化主Info Hash + ZSet榜单（仅首次执行）
func (s *ArticleRankService) initZSetCache(ctx context.Context) error {
	// 1. 获取文章详情
	articles, err := s.articleRepo.GetListByStatus(ctx, 1, model.Published)
	if err != nil {
		return err
	}
	if len(articles) == 0 {
		return nil
	}

	// 2. 开启管道
	pipe := s.rdb.Pipeline()
	zList := make([]redis.Z, 0, len(articles))

	// 3. 初始化文章info hash
	for _, art := range articles {
		hashKey := s.getArticleInfoHashKey(art.ID)
		// 冷启动一次性全量填充主Info Hash
		pipe.HSet(ctx, hashKey,
			"title", art.Title,
			"view_count", strconv.FormatUint(uint64(art.ViewCount), 10),
			"like_count", strconv.FormatUint(uint64(art.LikeCount), 10),
			"comment_count", strconv.FormatUint(uint64(art.CommentCount), 10),
		)
		pipe.Expire(ctx, hashKey, consts.ExpirePeriod)
		// 计算热度
		score := s.calcHotScore(art.ViewCount, art.LikeCount, art.CommentCount)
		member := strconv.FormatUint(art.ID, 10)
		zList = append(zList, redis.Z{Score: score, Member: member})
	}

	// 4. 创建热度ZSet榜单
	pipe.ZAdd(ctx, consts.KeyArticleHotRankZSet, zList...)
	pipe.ZRemRangeByRank(ctx, consts.KeyArticleHotRankZSet, 0, -101) // 限制榜单大小，防BigKey

	_, err = pipe.Exec(ctx)
	return err
}

// 全局冷启动初始化排行榜缓存
func (s *ArticleRankService) initRankCache(ctx context.Context) error {
	return cache.DoubleCheckInitCache(
		ctx,
		s.rdb,
		consts.KeyArticleHotRankZSet,
		consts.KeyArticleHotRankInitLock,
		"init",
		3*time.Second,
		3,
		100*time.Millisecond,
		s.checkZSetExists,
		s.initZSetCache,
	)
}

// 实时更新单篇文章热度分数（浏览/点赞/评论后调用，仅更新ZSet，不改动主Info Hash）
func (s *ArticleRankService) UpdateSingleArticleHot(ctx context.Context, articleID uint64) error {
	hashKey := s.getArticleInfoHashKey(articleID)
	res, err := s.rdb.HMGet(ctx, hashKey, "view_count", "like_count", "comment_count").Result()

	// 兜底：极端情况Hash不存在时查DB补数据（仅兜底，非主流程）
	if err != nil || len(res) < 3 || res[0] == nil || res[1] == nil || res[2] == nil {
		art, err := s.articleRepo.FindArticleByID(ctx, articleID)
		if err != nil {
			return err
		}
		score := s.calcHotScore(art.ViewCount, art.LikeCount, art.CommentCount)
		member := strconv.FormatUint(articleID, 10)
		_, err = s.rdb.ZAdd(ctx, consts.KeyArticleHotRankZSet, redis.Z{Score: score, Member: member}).Result()
		return err
	}

	var view, like, comment uint64
	if str, ok := res[0].(string); ok {
		view, _ = strconv.ParseUint(str, 10, 32)
	}
	if str, ok := res[1].(string); ok {
		like, _ = strconv.ParseUint(str, 10, 32)
	}
	if str, ok := res[2].(string); ok {
		comment, _ = strconv.ParseUint(str, 10, 32)
	}

	score := s.calcHotScore(uint32(view), uint32(like), uint32(comment))
	member := strconv.FormatUint(articleID, 10)
	_, err = s.rdb.ZAdd(ctx, consts.KeyArticleHotRankZSet, redis.Z{Score: score, Member: member}).Result()
	_ = s.rdb.ZRemRangeByRank(ctx, consts.KeyArticleHotRankZSet, 0, -101).Err()
	return err
}

// 获取前10热门文章榜单
func (s *ArticleRankService) GetTop10HotArticles(ctx context.Context) (*article.HotRankResponse, error) {

	// 1. 检查zset是否存在，不存在则拉取缓存
	if err := s.initRankCache(ctx); err != nil {
		return nil, err
	}

	// 2. 从ZSet倒序读取前10条热门文章（按热度分数降序）
	zRes, err := s.rdb.ZRevRangeWithScores(ctx, consts.KeyArticleHotRankZSet, 0, 9).Result()
	if err != nil {
		return nil, err
	}
	// 3. 组装返回结果数组
	result := make([]article.HotRankItem, 0, 10)
	for _, z := range zRes {
		// 读取ZSet中的文章ID字符串
		articleIDStr, ok := z.Member.(string)
		if !ok {
			continue
		}
		// 转为数字ID
		articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
		if err != nil {
			continue
		}
		// 拼接文章详情主Hash Key
		hashKey := s.getArticleInfoHashKey(articleID)
		// 读取文章基础信息和实时计数
		hashData, err := s.rdb.HMGet(ctx, hashKey, "title", "view_count", "like_count", "comment_count").Result()
		if err != nil || len(hashData) < 4 {
			continue
		}
		// 解析各个字段并转为数值类型
		title, _ := hashData[0].(string)
		viewCountStr, _ := hashData[1].(string)
		likeCountStr, _ := hashData[2].(string)
		commentCountStr, _ := hashData[3].(string)

		viewCount, _ := strconv.ParseUint(viewCountStr, 10, 64)
		likeCount, _ := strconv.ParseUint(likeCountStr, 10, 64)
		commentCount, _ := strconv.ParseUint(commentCountStr, 10, 64)
		// 构造单条榜单条目
		item := article.HotRankItem{
			ArticleID:    articleID,
			Title:        title,
			Hot:          z.Score,
			ViewCount:    uint32(viewCount),
			LikeCount:    uint32(likeCount),
			CommentCount: uint32(commentCount),
		}
		result = append(result, item)
	}
	resp := article.NewHotRankResponse(result)
	return resp, nil
}

// 每日兜底校准任务：以MySQL权威数据重算热度，修正Redis累计偏差 todo :删除
func (s *ArticleRankService) DailyCalibrate(ctx context.Context) error {
	// 1.读取全部有效已发布文章
	articles, err := s.articleRepo.GetListWithOffset(ctx, 1, 1000, false, 3)
	if err != nil {
		return err
	}
	// 2.批量更新Hash缓存和ZSet热度分数
	pipe := s.rdb.Pipeline()
	for _, art := range articles {
		hashKey := s.getArticleInfoHashKey(art.ID)
		pipe.HMSet(ctx, hashKey, map[string]any{
			"title":         art.Title,
			"view_count":    art.ViewCount,
			"like_count":    art.LikeCount,
			"comment_count": art.CommentCount,
		})
		score := s.calcHotScore(art.ViewCount, art.LikeCount, art.CommentCount)
		member := strconv.FormatUint(art.ID, 10)
		pipe.ZAdd(ctx, consts.KeyArticleHotRankZSet, redis.Z{Score: score, Member: member})
	}
	_, err = pipe.Exec(ctx)
	return err
}
