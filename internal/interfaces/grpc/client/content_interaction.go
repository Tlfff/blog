package client

import (
	domaincontent "blog/internal/domain/content"
	internalv1 "blog/shared/contracts/gen/internalv1"
	"context"
)

// ContentInteractionClient 通过 Community gRPC 为 Content 提供互动统计查询。
type ContentInteractionClient struct {
	client internalv1.CommunityServiceClient
}

func NewContentInteractionClient(client internalv1.CommunityServiceClient) *ContentInteractionClient {
	return &ContentInteractionClient{client: client}
}

func (c *ContentInteractionClient) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	resp, err := c.client.IsUserLikedArticle(ctx, &internalv1.ArticleUserRequest{
		ArticleId: articleID,
		UserId:    userID,
	})
	if err != nil {
		return false, toBizError(err)
	}
	return resp.Liked, nil
}

var _ domaincontent.ArticleInteractionQuery = (*ContentInteractionClient)(nil)
