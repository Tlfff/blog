package service

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/user"
	"blog/internal/repository"
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
func (s *UserService) GetMyProfile(userID uint64) (*user.MyProfileResponse, error) {
	u, err := s.repo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	return user.NewMyProfileResponse(u), nil
}

// 获取他人主页详情
func (s *UserService) GetUserProfile(userID uint64) (*user.UserProfileResponse, error) {
	u, err := s.repo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	return user.NewUserProfileResponse(u), nil
}

// 更新用户基本信息
func (s *UserService) UpdateProfile(userID uint64, nickname string, avatar string) error {
	u, err := s.repo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrUserNotFound
		}
		return err
	}

	u.Nickname = nickname
	u.Avatar = avatar

	return s.repo.UpdateUser(u)
}

// 更新用户密码
func (s *UserService) UpdatePassword(userID uint64, oldPassword string, newPassword string) error {
	u, err := s.repo.FindUserByID(userID)
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

	return s.repo.UpdateUser(u)
}

// 更新用户账户
func (s *UserService) UpdateAccount(userID uint64, phone string) error {
	u, err := s.repo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrUserNotFound
		}
		return err
	}

	// 检查新手机号是否已被他人占用
	existUser, err := s.repo.GetUserByAccount(phone)
	if err == nil && existUser.ID != userID {
		return common.ErrPhoneAlreadyExists
	}

	u.Phone = phone
	u.UpdatedTime = time.Now()

	return s.repo.UpdateUser(u)
}
