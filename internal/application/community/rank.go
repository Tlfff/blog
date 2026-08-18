package community

import (
	domaincommunity "blog/internal/domain/community"
	articledto "blog/internal/dto/article"
	"context"
)

// 获取文章热榜：从 Redis 取热度排名，再补齐文章标题与统计数据
func (s *Service) GetHotRank(ctx context.Context) (*articledto.HotRankResponse, error) {
	// 1. 从 Redis 热榜取前 10 名
	rankItems, err := s.hotRank.GetTop(ctx, 10)
	if err != nil {
		return nil, err
	}
	// 2. 榜单为空时返回空列表
	if len(rankItems) == 0 {
		return articledto.NewHotRankResponse([]articledto.HotRankItem{}), nil
	}
	// 3. 收集文章ID与对应热度分值
	ids := make([]uint64, 0, len(rankItems))
	scoreMap := make(map[uint64]float64, len(rankItems))
	for _, item := range rankItems {
		ids = append(ids, item.ArticleID)
		scoreMap[item.ArticleID] = item.Hot
	}
	// 4. 批量查询文章的标题与统计数据
	articles, err := s.articles.GetHotListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	// 5. 构建文章ID到文章信息的映射，便于按榜单顺序取值
	articleMap := make(map[uint64]*domaincommunity.ArticleInfo, len(articles))
	for _, article := range articles {
		articleMap[article.ID] = article
	}
	// 6. 按 Redis 榜单顺序组装响应，跳过已不存在的文章
	items := make([]articledto.HotRankItem, 0, len(ids))
	for _, id := range ids {
		article := articleMap[id]
		if article == nil {
			continue
		}
		items = append(items, articledto.HotRankItem{
			ArticleID:    id,
			Title:        article.Title,
			Hot:          scoreMap[id],
			ViewCount:    article.ViewCount,
			CommentCount: article.CommentCount,
			LikeCount:    article.LikeCount,
		})
	}
	return articledto.NewHotRankResponse(items), nil
}

// 重建文章热榜：按数据库统计重新计算热度并覆盖 Redis 榜单
func (s *Service) RebuildHotRank(ctx context.Context) error {
	// 1. 查询热度最高的前 100 篇已发表文章
	articles, err := s.articles.GetTopHotArticles(ctx, 100)
	if err != nil {
		return err
	}
	// 2. 逐条计算热度值，组装榜单条目
	entries := make([]domaincommunity.HotRankItem, 0, len(articles))
	for _, article := range articles {
		entries = append(entries, domaincommunity.HotRankItem{
			ArticleID:    article.ID,
			Title:        article.Title,
			Hot:          domaincommunity.CalcHotScore(article.ViewCount, article.LikeCount, article.CommentCount),
			ViewCount:    article.ViewCount,
			CommentCount: article.CommentCount,
			LikeCount:    article.LikeCount,
		})
	}
	// 3. 覆盖写入 Redis 榜单
	return s.hotRank.Rebuild(ctx, entries)
}
