package service

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/user"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

type UserAuthService struct {
	repo *repository.UserRepository
}

func NewUserAuthService(repo *repository.UserRepository) *UserAuthService {
	return &UserAuthService{repo: repo}
}

// 注册新用户
func (s *UserAuthService) Register(ctx context.Context, phone, password, nickname, clientIP string) error {
	// 1. 检查用户是否已存在
	_, err := s.repo.GetUserByAccount(ctx, phone, nickname)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 2. 密码哈希加密
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("密码哈希失败: %v", err)
		return common.ErrPasswordHashFailed
	}

	// 3. 在 Service 层完整组装并初始化 model 实体
	newUser := &model.User{
		Nickname:      nickname,
		Phone:         phone,
		Password:      passwordHash,
		Avatar:        "https://example.com/default-avatar.png", // 设定默认头像
		Role:          int8(model.RoleUser),
		Status:        1, // 1-正常
		LastLoginIp:   clientIP,
		LastLoginTime: time.Now(),
		CreatedTime:   time.Now(),
		UpdatedTime:   time.Now(),
	}

	log.Printf("Registering user: %+v\n", newUser)
	return s.repo.CreateUser(ctx, newUser)
}

// 登录验证
func (s *UserAuthService) Login(ctx context.Context, phone, nickname, password, clientIP string) (*user.LoginResponse, error) {

	// 1. 查找用户
	dbUser, err := s.repo.GetUserByAccount(ctx, phone, nickname)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	// 2. 验证密码
	ok, err := auth.VerifyPassword(password, dbUser.Password)
	if err != nil {
		log.Printf("密码验证失败: %v", err)
		return nil, common.ErrSystem
	}
	if !ok {
		log.Println("密码错误")
		return nil, common.ErrPasswordFailed
	}

	// 3. 更新登录信息
	dbUser.LastLoginIp = clientIP
	dbUser.LastLoginTime = time.Now()
	dbUser.UpdatedTime = time.Now()
	if err := s.repo.UpdateUser(ctx, dbUser); err != nil {
		// 登录中，更新 IP 失败通常不阻断登录，这里选择仅记录日志或非核心报错处理
		log.Printf("更新用户登录信息失败: %v", err)
	}

	// 4. 生成 JWT 令牌
	token, err := auth.GenerateToken(dbUser.Phone, dbUser.Role, dbUser.ID)
	if err != nil {
		log.Printf("生成 Token 失败: %v", err)
		return nil, common.ErrSystem
	}

	// 5. 返回组装好的 DTO 响应体
	return &user.LoginResponse{
		AccessToken: token,
	}, nil
}
