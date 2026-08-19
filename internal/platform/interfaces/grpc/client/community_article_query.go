package client

import (
	domainarticle "blog/internal/article/domain"
	internalv1 "blog/shared/contracts/gen/internalv1"
	"context"
)

// CommunityArticleQueryClient 通过 Content gRPC 提供文章基本信息查询，
// 热榜所需的统计列表查询继续委托给共享 MySQL 的本地只读适配器。
type CommunityArticleQueryClient struct {
	remote internalv1.ContentServiceClient
	local  domainarticle.RankingQuery
}

func NewCommunityArticleQueryClient(remote internalv1.ContentServiceClient, local domainarticle.RankingQuery) *CommunityArticleQueryClient {
	return &CommunityArticleQueryClient{remote: remote, local: local}
}

func (c *CommunityArticleQueryClient) FindByID(ctx context.Context, id uint64) (*domainarticle.ArticleInfo, error) {
	resp, err := c.remote.GetArticleInfo(ctx, &internalv1.ArticleIDRequest{ArticleId: id})
	if err != nil {
		return nil, toContentError(err)
	}
	return &domainarticle.ArticleInfo{
		ID:       resp.Id,
		AuthorID: resp.AuthorId,
		Title:    resp.Title,
	}, nil
}

func (c *CommunityArticleQueryClient) GetHotListByIDs(ctx context.Context, ids []uint64) ([]*domainarticle.ArticleInfo, error) {
	return c.local.GetHotListByIDs(ctx, ids)
}

func (c *CommunityArticleQueryClient) GetTopHotArticles(ctx context.Context, limit int) ([]*domainarticle.ArticleInfo, error) {
	return c.local.GetTopHotArticles(ctx, limit)
}
