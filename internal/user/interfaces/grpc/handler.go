package grpc

import (
	grpcinterface "blog/internal/platform/interfaces/grpc"
	userdto "blog/internal/user/app/dto"
	"context"

	userv1 "blog/gen/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserQueryUsecase 是开放 gRPC 用户查询的应用用例接口。
type UserQueryUsecase interface {
	// GetUserBasicInfo 查询用户基本信息。
	GetUserBasicInfo(ctx context.Context, userID uint64) (*userdto.UserBasicInfoResponse, error)
}

// UserHandler 实现开放 gRPC 用户服务。
type UserHandler struct {
	userv1.UnimplementedUserServiceServer                  // 防止编译错误
	userService                           UserQueryUsecase // 用户查询应用用例
}

// NewUserHandler 创建 User gRPC Handler。
func NewUserHandler(userService UserQueryUsecase) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetUserBasicInfo 获取用户基本信息。
func (h *UserHandler) GetUserBasicInfo(ctx context.Context, req *userv1.GetUserBasicInfoRequest) (*userv1.GetUserBasicInfoResponse, error) {
	// 1. 入参校验
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id必须大于0")
	}
	// 2. 获取用户信息
	info, err := h.userService.GetUserBasicInfo(ctx, req.UserId)
	if err != nil {
		return nil, grpcinterface.Error(err)
	}
	// 3. 构建返回响应
	return &userv1.GetUserBasicInfoResponse{
		UserId:        info.ID,
		Nickname:      info.Nickname,
		Avatar:        info.Avatar,
		LastLoginTime: info.LastLoginTime,
		LastLoginIp:   info.LastLoginIp,
	}, nil
}

// GetPublicUserInfo 获取用户公开信息。
func (h *UserHandler) GetPublicUserInfo(ctx context.Context, req *userv1.GetUserInfoRequest) (*userv1.GetUserInfoResponse, error) {
	// 1. 入参校验
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id必须大于0")
	}
	// 2. 复用二方相同的查询逻辑获取用户信息
	info, err := h.userService.GetUserBasicInfo(ctx, req.UserId)
	if err != nil {
		return nil, grpcinterface.Error(err)
	}
	// 3. 只映射公开字段
	return &userv1.GetUserInfoResponse{
		Id:       info.ID,
		Avatar:   info.Avatar,
		Nickname: info.Nickname,
	}, nil
}
