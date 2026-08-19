package client

import (
	domaincontent "blog/internal/article/domain"
	internalv1 "blog/shared/contracts/gen/internalv1"
	"context"
)

// ContentUserQueryClient 通过 Identity gRPC 为 Content 提供作者信息查询。
type ContentUserQueryClient struct {
	client internalv1.IdentityServiceClient
}

func NewContentUserQueryClient(client internalv1.IdentityServiceClient) *ContentUserQueryClient {
	return &ContentUserQueryClient{client: client}
}

func (c *ContentUserQueryClient) FindUserByID(ctx context.Context, id uint64) (*domaincontent.UserInfo, error) {
	resp, err := c.client.GetUserBasicInfo(ctx, &internalv1.UserIDRequest{UserId: id})
	if err != nil {
		return nil, toBizError(err)
	}
	return &domaincontent.UserInfo{
		ID:          resp.Id,
		Nickname:    resp.Nickname,
		Avatar:      resp.Avatar,
		LastLoginIP: resp.LastLoginIp,
	}, nil
}
