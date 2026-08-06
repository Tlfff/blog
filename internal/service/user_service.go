package service

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/user"
	"blog/internal/repository"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// 获取自己主页详情
func (s *UserService) GetMyProfile(ctx context.Context, userID uint64) (*user.MyProfileResponse, error) {
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	return user.NewMyProfileResponse(u), nil
}

// 获取他人主页详情
func (s *UserService) GetUserProfile(ctx context.Context, userID uint64) (*user.UserProfileResponse, error) {
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	return user.NewUserProfileResponse(u), nil
}

// 更新用户基本信息
func (s *UserService) UpdateProfile(ctx context.Context, userID uint64, nickname string, avatar string) error {
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrUserNotFound
		}
		return err
	}

	u.Nickname = nickname
	u.Avatar = avatar

	return s.repo.UpdateUser(ctx, u)
}

// 更新用户密码
func (s *UserService) UpdatePassword(ctx context.Context, userID uint64, oldPassword string, newPassword string) error {
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrUserNotFound
		}
		return err
	}

	ok, err := auth.VerifyPassword(oldPassword, u.Password)
	if err != nil {
		return err
	}
	if !ok {
		return common.ErrPasswordFailed
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	u.Password = hash
	u.UpdatedTime = time.Now()

	return s.repo.UpdateUser(ctx, u)
}

// 更新用户账户
func (s *UserService) UpdateAccount(ctx context.Context, userID uint64, phone string) error {
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrUserNotFound
		}
		return err
	}

	// 检查新手机号是否已被他人占用
	existUser, err := s.repo.GetUserByAccount(ctx, phone, "")
	if err == nil && existUser.ID != userID {
		return common.ErrPhoneAlreadyExists
	}

	u.Phone = phone
	u.UpdatedTime = time.Now()

	return s.repo.UpdateUser(ctx, u)
}

// -------------------------------------- 二方服务调用 --------------------------------------

// 获取用户基本信息
func (s *UserService) GetUserBasicInfo(ctx context.Context, userID uint64) (*user.UserBasicInfoResponse, error) {
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	return user.NewUserBasicInfoResponse(u), nil
}

// 获取用户基本信息列表
func (s *UserService) ListUserBasicInfo(ctx context.Context, page, pageSize uint64, isDesc bool) (*user.UserListResponse, error) {
	// 1. 获取用户列表
	users, err := s.repo.GetUserList(ctx, int(page), int(pageSize), isDesc)
	if err != nil {
		return nil, err
	}
	// 2. 获取总用户数
	total, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	// 3. 构建返回响应
	resp := user.NewUserListResponse(users, uint64(total), page, pageSize)

	return resp, nil
}
