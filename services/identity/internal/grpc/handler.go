// Package grpc 实现 Identity Service 的内部 gRPC 接口。
package grpc

import (
	identityapp "blog/internal/application/identity"
	userdto "blog/internal/dto/user"
	internalv1 "blog/shared/contracts/gen/internalv1"
	platformerrors "blog/shared/platform/errors"
	"context"
)

// IdentityServer 把 Identity Application 用例暴露为内部 gRPC 服务。
type IdentityServer struct {
	internalv1.UnimplementedIdentityServiceServer
	app *identityapp.Service
}

func NewIdentityServer(app *identityapp.Service) *IdentityServer {
	return &IdentityServer{app: app}
}

func (s *IdentityServer) Register(ctx context.Context, req *internalv1.RegisterRequest) (*internalv1.Empty, error) {
	if err := s.app.Register(ctx, req.Phone, req.Password, req.Nickname, req.ClientIp); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *IdentityServer) Login(ctx context.Context, req *internalv1.LoginRequest) (*internalv1.LoginResponse, error) {
	resp, err := s.app.Login(ctx, req.Phone, req.Nickname, req.Password, req.ClientIp, req.Device, req.RememberMe)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.LoginResponse{AccessToken: resp.AccessToken}, nil
}

func (s *IdentityServer) Logout(ctx context.Context, req *internalv1.LogoutRequest) (*internalv1.Empty, error) {
	if err := s.app.Logout(ctx, req.Token); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *IdentityServer) GetMyProfile(ctx context.Context, req *internalv1.UserIDRequest) (*internalv1.MyProfile, error) {
	profile, err := s.app.GetMyProfile(ctx, req.UserId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.MyProfile{
		Id:            profile.ID,
		Nickname:      profile.Nickname,
		Avatar:        profile.Avatar,
		Role:          int32(profile.Role),
		LastLoginTime: profile.LastLoginTime,
		LastLoginIp:   profile.LastLoginIp,
	}, nil
}

func (s *IdentityServer) GetPublicProfile(ctx context.Context, req *internalv1.UserIDRequest) (*internalv1.PublicProfile, error) {
	profile, err := s.app.GetUserProfile(ctx, req.UserId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.PublicProfile{
		Id:       profile.ID,
		Nickname: profile.Nickname,
		Avatar:   profile.Avatar,
	}, nil
}

func (s *IdentityServer) UpdateProfile(ctx context.Context, req *internalv1.UpdateProfileRequest) (*internalv1.Empty, error) {
	if err := s.app.UpdateProfile(ctx, req.UserId, req.Nickname, req.Avatar); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *IdentityServer) VerifyOldPassword(ctx context.Context, req *internalv1.VerifyOldPasswordRequest) (*internalv1.VerifyOldPasswordResponse, error) {
	token, err := s.app.VerifyOldPassword(ctx, req.UserId, req.OldPassword)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.VerifyOldPasswordResponse{ChangeToken: token}, nil
}

func (s *IdentityServer) ChangePassword(ctx context.Context, req *internalv1.ChangePasswordRequest) (*internalv1.Empty, error) {
	if err := s.app.ChangePassword(ctx, req.UserId, req.ChangeToken, req.NewPassword, req.CurrentToken); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *IdentityServer) UpdatePassword(ctx context.Context, req *internalv1.UpdatePasswordRequest) (*internalv1.Empty, error) {
	if err := s.app.UpdatePassword(ctx, req.UserId, req.OldPassword, req.NewPassword); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *IdentityServer) UpdateAccount(ctx context.Context, req *internalv1.UpdateAccountRequest) (*internalv1.Empty, error) {
	if err := s.app.UpdateAccount(ctx, req.UserId, req.Phone); err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.Empty{}, nil
}

func (s *IdentityServer) GetAvatarUploadURL(ctx context.Context, req *internalv1.GetAvatarUploadURLRequest) (*internalv1.UploadURLResponse, error) {
	uploadURL, objectKey, err := s.app.GetAvatarUploadURL(ctx, req.UserId, req.FileExt)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.UploadURLResponse{UploadUrl: uploadURL, ObjectKey: objectKey}, nil
}

func (s *IdentityServer) ConfirmAvatar(ctx context.Context, req *internalv1.ConfirmAvatarRequest) (*internalv1.ConfirmAvatarResponse, error) {
	avatarURL, err := s.app.ConfirmAvatar(ctx, req.UserId, req.ObjectKey)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.ConfirmAvatarResponse{AvatarUrl: avatarURL}, nil
}

func (s *IdentityServer) GetUserBasicInfo(ctx context.Context, req *internalv1.UserIDRequest) (*internalv1.UserBasicInfo, error) {
	info, err := s.app.GetUserBasicInfo(ctx, req.UserId)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	return &internalv1.UserBasicInfo{
		Id:            info.ID,
		Nickname:      info.Nickname,
		Avatar:        info.Avatar,
		LastLoginTime: info.LastLoginTime,
		LastLoginIp:   info.LastLoginIp,
	}, nil
}

func (s *IdentityServer) ListUserBasicInfo(ctx context.Context, req *internalv1.ListUsersRequest) (*internalv1.ListUsersResponse, error) {
	resp, err := s.app.ListUserBasicInfo(ctx, req.Page, req.PageSize, req.IsDesc)
	if err != nil {
		return nil, platformerrors.ToGRPC(err)
	}
	items := make([]*internalv1.UserListItem, 0, len(resp.List))
	for _, item := range resp.List {
		items = append(items, &internalv1.UserListItem{
			Id:       item.ID,
			Nickname: item.Nickname,
			Avatar:   item.Avatar,
		})
	}
	return &internalv1.ListUsersResponse{
		Items:    items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

var _ = userdto.LoginResponse{}
