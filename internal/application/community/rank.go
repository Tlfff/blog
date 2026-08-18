package community

import (
	domaincommunity "blog/internal/domain/community"
	articledto "blog/internal/dto/article"
	"context"
)

func (s *Service) GetHotRank(ctx context.Context) (*articledto.HotRankResponse, error) {
	rankItems, err := s.hotRank.GetTop(ctx, 10)
	if err != nil {
		return nil, err
	}
	if len(rankItems) == 0 {
		return articledto.NewHotRankResponse([]articledto.HotRankItem{}), nil
	}
	ids := make([]uint64, 0, len(rankItems))
	scoreMap := make(map[uint64]float64, len(rankItems))
	for _, item := range rankItems {
		ids = append(ids, item.ArticleID)
		scoreMap[item.ArticleID] = item.Hot
	}
	articles, err := s.articles.GetHotListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	articleMap := make(map[uint64]*domaincommunity.ArticleInfo, len(articles))
	for _, article := range articles {
		articleMap[article.ID] = article
	}
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

func (s *Service) RebuildHotRank(ctx context.Context) error {
	articles, err := s.articles.GetTopHotArticles(ctx, 100)
	if err != nil {
		return err
	}
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
	return s.hotRank.Rebuild(ctx, entries)
}
