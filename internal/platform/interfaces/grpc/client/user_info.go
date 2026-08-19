package client

import (
	domainnotification "blog/internal/notification/domain"
	internalv1 "blog/shared/contracts/gen/internalv1"
	"context"
)

// CommunityUserInfoClient 通过 Identity gRPC 为 Community 提供用户公开信息查询。
type CommunityUserInfoClient struct {
	client internalv1.IdentityServiceClient
}

func NewCommunityUserInfoClient(client internalv1.IdentityServiceClient) *CommunityUserInfoClient {
	return &CommunityUserInfoClient{client: client}
}

func (c *CommunityUserInfoClient) FindUserByID(ctx context.Context, id uint64) (*domainnotification.UserInfo, error) {
	resp, err := c.client.GetUserBasicInfo(ctx, &internalv1.UserIDRequest{UserId: id})
	if err != nil {
		return nil, toBizError(err)
	}
	return &domainnotification.UserInfo{
		ID:       resp.Id,
		Nickname: resp.Nickname,
		Avatar:   resp.Avatar,
	}, nil
}
