package service

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
	"time"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// 获取用户详情
func (s *UserService) GetProfile(userID int64) (*model.User, error) {
	return s.repo.FindUserByID(userID)
}

// 更新用户基本信息
func (s *UserService) UpdateProfile(userID int64, nickname string, avatar string) error {

	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	user.Nickname = nickname
	user.Avatar = avatar
	user.UpdateTime = time.Now()

	return s.repo.UpdateUser(user)
}

// 更新用户账户
func (s *UserService) UpdateAccount(userID int64, phone string, oldPassword string, newPassword string) error {

	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	ok, err := auth.VerifyPassword(
		oldPassword,
		user.Password,
	)

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

	user.Phone = phone
	user.Password = hash
	user.UpdateTime = time.Now()

	return s.repo.UpdateUser(user)
}
