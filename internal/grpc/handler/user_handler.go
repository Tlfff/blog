package handler

import (
	"blog/internal/service"
	"context"

	userv1 "blog/gen/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 实现二方服务的用户接口

type UserHandler struct {
	userv1.UnimplementedUserServiceServer // 防止编译错误
	userService                           *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// 获取用户基本信息（头像、昵称、最后登录信息）—— 二方服务
func (h *UserHandler) GetUserBasicInfo(ctx context.Context, req *userv1.GetUserBasicInfoRequest) (*userv1.GetUserBasicInfoResponse, error) {
	// 1. 入参校验
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id必须大于0")
	}
	// 2. 获取用户信息
	info, err := h.userService.GetUserBasicInfo(ctx, req.UserId)
	if err != nil {
		return nil, GRPCError(err)
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

// 获取用户公开信息（ID、头像、昵称）—— 三方合作方服务
func (h *UserHandler) GetPublicUserInfo(ctx context.Context, req *userv1.GetUserInfoRequest) (*userv1.GetUserInfoResponse, error) {
	// 1. 入参校验
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id必须大于0")
	}
	// 2. 复用二方相同的查询逻辑获取用户信息
	info, err := h.userService.GetUserBasicInfo(ctx, req.UserId)
	if err != nil {
		return nil, GRPCError(err)
	}
	// 3. 只映射公开字段
	return &userv1.GetUserInfoResponse{
		Id:       info.ID,
		Avatar:   info.Avatar,
		Nickname: info.Nickname,
	}, nil
}
