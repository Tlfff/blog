package service

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
	"errors"
	"log"
	"time"
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
	_, err = s.repo.GetUserByAccount(user.Phone)
	if err == nil {
		log.Println("用户已存在")
		return errors.New(common.ErrUserExists.Error())
	}
	user.Password = passwordHash
	user.Status = 1
	return s.repo.CreateUser(user)
}

// 登录验证
func (s *UserAuthService) Login(Account, Password string) (*model.User, error) {
	log.Printf("Attempting login for phone: %s\n", Account)
	user, err := s.repo.GetUserByAccount(Account)
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
		log.Printf("用户%s不存在或已被禁用", Account)
		return nil, errors.New(common.ErrUserNotFound.Error())
	}
	return user, nil
}

// 更新用户登录信息
func (s *UserAuthService) UpdateLoginInfo(userID int64, ip string, loginTime time.Time) error {

	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	user.LastLoginIp = ip
	user.LastLoginTime = loginTime

	return s.repo.UpdateUser(user)
}
