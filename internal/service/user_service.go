package service

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/user"
	"blog/internal/repository"
	"blog/pkg/oss"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	passwordChangeTokenPrefix = "user:password-change:"
	passwordChangeTokenTTL    = 10 * time.Minute
)

type UserService struct {
	repo            *repository.UserRepository
	rdb             *redis.Client
	oss             *oss.MinioClient
	ossPublicDomain string
	allowedExts     map[string]bool // 允许上传的头像扩展名
}

// NewUserService 构造函数；rdb 为可选依赖（可变参数），
// 某些调用场景（gRPC 服务、单元测试）用不到密码修改凭证等 Redis 功能，可不传
func NewUserService(repo *repository.UserRepository, rdbs ...*redis.Client) *UserService {
	var rdb *redis.Client
	if len(rdbs) > 0 {
		rdb = rdbs[0]
	}
	return &UserService{repo: repo, rdb: rdb}
}

// SetOSS 注入 MinIO 客户端、公开域名和允许的扩展名
// OSS 为可选依赖：gRPC 等二方服务用不到 OSS 功能，无需注入（相关接口会返回系统异常）
func (s *UserService) SetOSS(ossClient *oss.MinioClient, publicDomain string, allowedExts []string) {
	s.oss = ossClient
	s.ossPublicDomain = publicDomain
	s.allowedExts = make(map[string]bool, len(allowedExts))
	for _, ext := range allowedExts {
		s.allowedExts[strings.ToLower(ext)] = true
	}
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

// 验证旧密码并生成一次性修改凭证
func (s *UserService) VerifyOldPassword(ctx context.Context, userID uint64, oldPassword string) (string, error) {
	if s.rdb == nil {
		return "", common.ErrSystem
	}

	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", common.ErrUserNotFound
		}
		return "", err
	}

	ok, err := auth.VerifyPassword(oldPassword, u.Password)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", common.ErrPasswordFailed
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", err
	}
	token := hex.EncodeToString(rawToken)
	if err := s.rdb.Set(ctx, passwordChangeTokenPrefix+token, userID, passwordChangeTokenTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// 兼容旧调用方的单接口密码修改方法，HTTP 路由不再使用
func (s *UserService) UpdatePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
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

// 使用一次性凭证修改密码
func (s *UserService) ChangePassword(ctx context.Context, userID uint64, changeToken, newPassword, currentToken string) error {
	if s.rdb == nil {
		return common.ErrSystem
	}

	value, err := s.rdb.GetDel(ctx, passwordChangeTokenPrefix+changeToken).Result()
	if errors.Is(err, redis.Nil) {
		return common.ErrPasswordChangeToken
	}
	if err != nil {
		return err
	}
	if value != fmt.Sprint(userID) {
		return common.ErrPasswordChangeToken
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrUserNotFound
		}
		return err
	}
	u.Password = hash
	u.UpdatedTime = time.Now()
	if err := s.repo.UpdateUser(ctx, u); err != nil {
		return err
	}

	// 强制其他设备下线，保留当前设备
	tokenAuth := auth.NewTokenAuth(s.rdb)
	return tokenAuth.DeleteOtherSessions(ctx, userID, currentToken)
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

// 获取头像上传凭证
func (s *UserService) GetAvatarUploadURL(ctx context.Context, userID uint64, fileExt string) (uploadURL, objectKey string, err error) {
	if s.oss == nil {
		return "", "", common.ErrSystem
	}

	// 校验文件扩展名白名单（来自配置）
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	if !s.allowedExts[ext] {
		return "", "", common.ErrInvalidRequestBody
	}

	// 生成 object key: avatar/{user_id}/{uuid}.{ext}
	objectKey = path.Join("avatar", fmt.Sprint(userID), uuid.NewString()+"."+ext)

	// 生成预签名 PUT URL，有效期 10 分钟
	uploadURL, err = s.oss.PresignedPutURL(ctx, objectKey, 10*time.Minute)
	if err != nil {
		return "", "", err
	}

	return uploadURL, objectKey, nil
}

// 确认头像上传完成
func (s *UserService) ConfirmAvatar(ctx context.Context, userID uint64, objectKey string) (avatarURL string, err error) {
	if s.oss == nil {
		return "", common.ErrSystem
	}

	// 校验 key 前缀，防止篡改他人头像
	expectedPrefix := "avatar/" + fmt.Sprint(userID) + "/"
	if !strings.HasPrefix(objectKey, expectedPrefix) {
		return "", common.ErrInvalidRequestBody
	}

	// 更新数据库
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", common.ErrUserNotFound
		}
		return "", err
	}
	u.Avatar = objectKey
	u.UpdatedTime = time.Now()
	if err := s.repo.UpdateUser(ctx, u); err != nil {
		return "", err
	}

	// 返回完整访问 URL
	avatarURL = s.oss.GetObjectURL(s.ossPublicDomain, objectKey)
	return avatarURL, nil
}
