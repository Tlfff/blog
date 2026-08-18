// Package client 提供统一入口到各服务内部 gRPC 接口的客户端适配。
package client

import (
	"blog/internal/common"
	userdto "blog/internal/dto/user"
	internalv1 "blog/shared/contracts/gen/internalv1"
	platformerrors "blog/shared/platform/errors"
	"context"
)

// IdentityClient 实现 HTTP/gRPC 入口需要的 Identity 用例接口。
type IdentityClient struct {
	client internalv1.IdentityServiceClient
}

func NewIdentityClient(client internalv1.IdentityServiceClient) *IdentityClient {
	return &IdentityClient{client: client}
}

func (c *IdentityClient) Register(ctx context.Context, phone, password, nickname, clientIP string) error {
	_, err := c.client.Register(ctx, &internalv1.RegisterRequest{
		Phone:    phone,
		Password: password,
		Nickname: nickname,
		ClientIp: clientIP,
	})
	return toBizError(err)
}

func (c *IdentityClient) Login(ctx context.Context, phone, nickname, password, clientIP, device string, rememberMe bool) (*userdto.LoginResponse, error) {
	resp, err := c.client.Login(ctx, &internalv1.LoginRequest{
		Phone:      phone,
		Nickname:   nickname,
		Password:   password,
		ClientIp:   clientIP,
		Device:     device,
		RememberMe: rememberMe,
	})
	if err != nil {
		return nil, toBizError(err)
	}
	return &userdto.LoginResponse{AccessToken: resp.AccessToken}, nil
}

func (c *IdentityClient) Logout(ctx context.Context, token string) error {
	_, err := c.client.Logout(ctx, &internalv1.LogoutRequest{Token: token})
	return toBizError(err)
}

func (c *IdentityClient) GetMyProfile(ctx context.Context, userID uint64) (*userdto.MyProfileResponse, error) {
	resp, err := c.client.GetMyProfile(ctx, &internalv1.UserIDRequest{UserId: userID})
	if err != nil {
		return nil, toBizError(err)
	}
	return &userdto.MyProfileResponse{
		ID:            resp.Id,
		Nickname:      resp.Nickname,
		Avatar:        resp.Avatar,
		Role:          int8(resp.Role),
		LastLoginTime: resp.LastLoginTime,
		LastLoginIp:   resp.LastLoginIp,
	}, nil
}

func (c *IdentityClient) GetUserProfile(ctx context.Context, userID uint64) (*userdto.UserProfileResponse, error) {
	resp, err := c.client.GetPublicProfile(ctx, &internalv1.UserIDRequest{UserId: userID})
	if err != nil {
		return nil, toBizError(err)
	}
	return &userdto.UserProfileResponse{
		ID:       resp.Id,
		Nickname: resp.Nickname,
		Avatar:   resp.Avatar,
	}, nil
}

func (c *IdentityClient) UpdateProfile(ctx context.Context, userID uint64, nickname, avatar string) error {
	_, err := c.client.UpdateProfile(ctx, &internalv1.UpdateProfileRequest{
		UserId:   userID,
		Nickname: nickname,
		Avatar:   avatar,
	})
	return toBizError(err)
}

func (c *IdentityClient) VerifyOldPassword(ctx context.Context, userID uint64, oldPassword string) (string, error) {
	resp, err := c.client.VerifyOldPassword(ctx, &internalv1.VerifyOldPasswordRequest{
		UserId:      userID,
		OldPassword: oldPassword,
	})
	if err != nil {
		return "", toBizError(err)
	}
	return resp.ChangeToken, nil
}

func (c *IdentityClient) UpdatePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	_, err := c.client.UpdatePassword(ctx, &internalv1.UpdatePasswordRequest{
		UserId:      userID,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	})
	return toBizError(err)
}

func (c *IdentityClient) ChangePassword(ctx context.Context, userID uint64, changeToken, newPassword, currentToken string) error {
	_, err := c.client.ChangePassword(ctx, &internalv1.ChangePasswordRequest{
		UserId:       userID,
		ChangeToken:  changeToken,
		NewPassword:  newPassword,
		CurrentToken: currentToken,
	})
	return toBizError(err)
}

func (c *IdentityClient) UpdateAccount(ctx context.Context, userID uint64, phone string) error {
	_, err := c.client.UpdateAccount(ctx, &internalv1.UpdateAccountRequest{UserId: userID, Phone: phone})
	return toBizError(err)
}

func (c *IdentityClient) GetAvatarUploadURL(ctx context.Context, userID uint64, fileExt string) (uploadURL, objectKey string, err error) {
	resp, err := c.client.GetAvatarUploadURL(ctx, &internalv1.GetAvatarUploadURLRequest{
		UserId:  userID,
		FileExt: fileExt,
	})
	if err != nil {
		return "", "", toBizError(err)
	}
	return resp.UploadUrl, resp.ObjectKey, nil
}

func (c *IdentityClient) ConfirmAvatar(ctx context.Context, userID uint64, objectKey string) (string, error) {
	resp, err := c.client.ConfirmAvatar(ctx, &internalv1.ConfirmAvatarRequest{
		UserId:    userID,
		ObjectKey: objectKey,
	})
	if err != nil {
		return "", toBizError(err)
	}
	return resp.AvatarUrl, nil
}

func (c *IdentityClient) GetUserBasicInfo(ctx context.Context, userID uint64) (*userdto.UserBasicInfoResponse, error) {
	resp, err := c.client.GetUserBasicInfo(ctx, &internalv1.UserIDRequest{UserId: userID})
	if err != nil {
		return nil, toBizError(err)
	}
	return &userdto.UserBasicInfoResponse{
		ID:            resp.Id,
		Nickname:      resp.Nickname,
		Avatar:        resp.Avatar,
		LastLoginTime: resp.LastLoginTime,
		LastLoginIp:   resp.LastLoginIp,
	}, nil
}

func toBizError(err error) error {
	if err == nil {
		return nil
	}
	switch platformerrors.FromGRPC(err) {
	case common.CodeUserNotFound:
		return common.ErrUserNotFound
	case common.CodeForbidden:
		return common.ErrForbidden
	case common.CodeUnauthorized:
		return common.ErrTokenInvalid
	case common.CodeInvalidParameter:
		return common.ErrParameter
	default:
		return common.ErrSystem
	}
}
