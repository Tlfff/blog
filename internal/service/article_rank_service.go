package service

import (
	"blog/internal/consts"
	"blog/internal/dto/article"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"log"
	"strconv"

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

// 检查排行榜ZSet是否存在
func (s *ArticleRankService) checkZSetExists(ctx context.Context) (bool, error) {
	exists, err := s.rdb.Exists(ctx, consts.KeyArticleHotRankZSet).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// 获取前10热门文章榜单
func (s *ArticleRankService) GetHotRank(ctx context.Context) (*article.HotRankResponse, error) {
	// 1.获取前10的文章
	result, err := s.rdb.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:   consts.KeyArticleHotRankZSet,
		Start: 0,
		Stop:  9,
		Rev:   true, // 降序
	}).Result()

	if err != nil {
		return nil, err
	}
	// 没文章则返回空
	if len(result) == 0 {
		log.Printf("排行榜单为空")
		return article.NewHotRankResponse([]article.HotRankItem{}), nil
	}

	// 2. 解析出id列表,同时记录每个id对应的热度分数,方便后面组装
	ids := make([]uint64, 0, len(result))
	scoreMap := make(map[uint64]float64, len(result))
	for _, z := range result {
		log.Printf("[debug]: member=%v, type=%T, score=%v", z.Member, z.Member, z.Score)
		idStr, ok := z.Member.(string)
		if !ok {
			log.Printf("排行榜member类型异常: %v", z.Member)
			continue
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			log.Printf("解析排行榜文章id失败,idStr:%s,err:%v", idStr, err)
			continue
		}
		ids = append(ids, id)
		scoreMap[id] = z.Score
	}

	// 3. 一次性批量查询这些文章的详情
	articles, err := s.articleRepo.GetHotListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	log.Printf("debug: articles=%+v", articles)
	// 4. 把文章详情按id放入map,方便按zset的顺序重新组装(IN查询不保证返回顺序)
	articleMap := make(map[uint64]*model.Article, len(articles))
	for _, a := range articles {
		articleMap[a.ID] = a
	}
	log.Printf("debug: articleMap len=%d", len(articleMap))
	// 5. 按zset原本的顺序(热度从高到低)组装最终返回结果
	items := make([]article.HotRankItem, 0, len(ids))
	for _, id := range ids {
		a, ok := articleMap[id]
		log.Printf("debug: id=%d, found=%v", id, ok)
		if !ok {
			// 极端情况:zset里有这个id,但文章可能已被删除,跳过即可
			continue
		}
		items = append(items, article.HotRankItem{
			ArticleID:    id,
			Title:        a.Title,
			Hot:          scoreMap[id],
			ViewCount:    a.ViewCount,
			CommentCount: a.CommentCount,
			LikeCount:    a.LikeCount,
		})
	}
	return article.NewHotRankResponse(items), nil
}

// 全量重建排行榜,供启动初始化和定时任务共用
func (s *ArticleRankService) RebuildHotRank(ctx context.Context) error {
	// 1. 获取排行前100文章
	articles, err := s.articleRepo.GetTopHotArticles(ctx, 100)
	if err != nil {
		return err
	}

	if len(articles) == 0 {
		return nil
	}

	zs := make([]redis.Z, 0, len(articles))
	for _, a := range articles {
		heat := calcHotScore(a.ViewCount, a.LikeCount, a.CommentCount)
		zs = append(zs, redis.Z{Score: heat, Member: a.ID})
	}

	// 2. 用pipeline把"清空旧数据"和"写入新数据"打包成一次原子操作
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, consts.KeyArticleHotRankZSet)
	pipe.ZAdd(ctx, consts.KeyArticleHotRankZSet, zs...)
	_, err = pipe.Exec(ctx)
	return err
}
