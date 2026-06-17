package service

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
	"errors"
	"log"
)

type UserAuthService struct {
	repo *repository.UserRepository
}

func NewUserAuthService(repo *repository.UserRepository) *UserAuthService {
	return &UserAuthService{repo: repo}
}

// 注册新用户
func (s *UserAuthService) Register(user *model.User, password string) error {
	log.Printf("Registering user: %+v\n", user)
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("密码哈希失败: %v", err)
		return errors.New(common.ErrPasswordHashFailed.Error())
	}
	_, err = s.repo.GetUserByPhone(user.Phone)
	if err == nil {
		log.Println("用户已存在")
		return errors.New(common.ErrUserExists.Error())
	}
	user.Password = passwordHash
	user.Status = 1
	return s.repo.CreateUser(user)
}

// 登录验证
func (s *UserAuthService) Login(Phone, Password string) (*model.User, error) {
	log.Printf("Attempting login for phone: %s\n", Phone)
	user, err := s.repo.GetUserByPhone(Phone)
	if err != nil {
		return nil, err
	}
	ok, err := auth.VerifyPassword(Password, user.Password)
	if err != nil {
		log.Printf("密码验证失败: %v", err)
		return nil, errors.New(common.ErrTokenInvalid.Error())
	}
	if !ok {
		log.Println("密码错误")
		return nil, errors.New(common.ErrPasswordFailed.Error())
	}
	if user.Status != 1 {
		log.Printf("用户%s不存在或已被禁用", Phone)
		return nil, errors.New(common.ErrUserNotFound.Error())
	}
	return user, nil
}
