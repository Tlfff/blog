// Package identity 提供 Identity 领域的应用用例。
package identity

import (
	"blog/internal/common"
	domainidentity "blog/internal/domain/identity"
	userdto "blog/internal/dto/user"
	iputil "blog/pkg/util/ip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	users          domainidentity.UserRepository
	sessions       domainidentity.TokenSession
	passwordTokens domainidentity.PasswordChangeTokenStore
	avatars        domainidentity.AvatarObjectStorage
	publicDomain   string
	allowedExts    map[string]bool
}

// NewService 组装 Identity Application 用例依赖。
func NewService(
	users domainidentity.UserRepository,
	sessions domainidentity.TokenSession,
	passwordTokens domainidentity.PasswordChangeTokenStore,
	avatars domainidentity.AvatarObjectStorage,
	publicDomain string,
	allowedExts []string,
) *Service {
	extMap := make(map[string]bool, len(allowedExts))
	for _, ext := range allowedExts {
		extMap[strings.ToLower(ext)] = true
	}
	return &Service{
		users:          users,
		sessions:       sessions,
		passwordTokens: passwordTokens,
		avatars:        avatars,
		publicDomain:   publicDomain,
		allowedExts:    extMap,
	}
}

func (s *Service) Register(ctx context.Context, phone, password, nickname, clientIP string) error {
	_, err := s.users.GetUserByAccount(ctx, phone, nickname)
	if err != nil && !errors.Is(err, domainidentity.ErrUserNotFound) {
		return err
	}

	passwordHash, err := domainidentity.HashPassword(password)
	if err != nil {
		log.Printf("密码哈希失败: %v", err)
		return common.ErrPasswordHashFailed
	}

	now := time.Now()
	user := &domainidentity.User{
		Nickname:      nickname,
		Phone:         phone,
		Password:      passwordHash,
		Avatar:        "https://example.com/default-avatar.png",
		Role:          domainidentity.RoleUser,
		Status:        domainidentity.StatusNormal,
		LastLoginIP:   clientIP,
		LastLoginTime: now,
		CreatedTime:   now,
		UpdatedTime:   now,
	}
	return s.users.CreateUser(ctx, user)
}

func (s *Service) Login(ctx context.Context, phone, nickname, password, clientIP, device string, rememberMe bool) (*userdto.LoginResponse, error) {
	user, err := s.users.GetUserByAccount(ctx, phone, nickname)
	if err != nil {
		if errors.Is(err, domainidentity.ErrUserNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	ok, err := domainidentity.VerifyPassword(password, user.Password)
	if err != nil {
		log.Printf("密码验证失败: %v", err)
		return nil, common.ErrSystem
	}
	if !ok {
		return nil, common.ErrPasswordFailed
	}

	user.LastLoginIP = clientIP
	user.LastLoginTime = time.Now()
	user.UpdatedTime = time.Now()
	if err := s.users.UpdateUser(ctx, user); err != nil {
		log.Printf("更新用户登录信息失败: %v", err)
	}

	if s.sessions == nil {
		return nil, common.ErrSystem
	}
	token, err := s.sessions.CreateSession(ctx, user.ID, user.Role, clientIP, device, rememberMe)
	if err != nil {
		log.Printf("创建登录会话失败: %v", err)
		return nil, common.ErrSystem
	}
	return &userdto.LoginResponse{AccessToken: token}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if s.sessions == nil {
		return common.ErrSystem
	}
	return s.sessions.DeleteSession(ctx, token)
}

func (s *Service) GetMyProfile(ctx context.Context, userID uint64) (*userdto.MyProfileResponse, error) {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &userdto.MyProfileResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Role:          user.Role,
		LastLoginTime: user.LastLoginTime.Unix(),
		LastLoginIp:   iputil.ConvertIPToRegion(user.LastLoginIP),
	}, nil
}

func (s *Service) GetUserProfile(ctx context.Context, userID uint64) (*userdto.UserProfileResponse, error) {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &userdto.UserProfileResponse{
		ID:       user.ID,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uint64, nickname, avatar string) error {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	user.Nickname = nickname
	user.Avatar = avatar
	user.UpdatedTime = time.Now()
	return s.users.UpdateUser(ctx, user)
}

func (s *Service) VerifyOldPassword(ctx context.Context, userID uint64, oldPassword string) (string, error) {
	if s.passwordTokens == nil {
		return "", common.ErrSystem
	}
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return "", err
	}
	ok, err := domainidentity.VerifyPassword(oldPassword, user.Password)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", common.ErrPasswordFailed
	}
	return s.passwordTokens.Issue(ctx, userID)
}

func (s *Service) UpdatePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	ok, err := domainidentity.VerifyPassword(oldPassword, user.Password)
	if err != nil {
		return err
	}
	if !ok {
		return common.ErrPasswordFailed
	}
	hash, err := domainidentity.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.Password = hash
	user.UpdatedTime = time.Now()
	return s.users.UpdateUser(ctx, user)
}

func (s *Service) ChangePassword(ctx context.Context, userID uint64, changeToken, newPassword, currentToken string) error {
	if s.passwordTokens == nil {
		return common.ErrSystem
	}
	tokenUserID, err := s.passwordTokens.Consume(ctx, changeToken)
	if errors.Is(err, domainidentity.ErrPasswordChangeToken) {
		return common.ErrPasswordChangeToken
	}
	if err != nil {
		return err
	}
	if tokenUserID != userID {
		return common.ErrPasswordChangeToken
	}

	hash, err := domainidentity.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	user.Password = hash
	user.UpdatedTime = time.Now()
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return err
	}
	if s.sessions != nil {
		return s.sessions.DeleteOtherSessions(ctx, userID, currentToken)
	}
	return nil
}

func (s *Service) UpdateAccount(ctx context.Context, userID uint64, phone string) error {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}

	existing, err := s.users.GetUserByAccount(ctx, phone, "")
	if err == nil && existing.ID != userID {
		return common.ErrPhoneAlreadyExists
	}
	if err != nil && !errors.Is(err, domainidentity.ErrUserNotFound) {
		return err
	}

	user.Phone = phone
	user.UpdatedTime = time.Now()
	return s.users.UpdateUser(ctx, user)
}

func (s *Service) GetUserBasicInfo(ctx context.Context, userID uint64) (*userdto.UserBasicInfoResponse, error) {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &userdto.UserBasicInfoResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		LastLoginTime: user.LastLoginTime.Unix(),
		LastLoginIp:   iputil.ConvertIPToRegion(user.LastLoginIP),
	}, nil
}

func (s *Service) ListUserBasicInfo(ctx context.Context, page, pageSize uint64, isDesc bool) (*userdto.UserListResponse, error) {
	users, err := s.users.GetUserList(ctx, int(page), int(pageSize), isDesc)
	if err != nil {
		return nil, err
	}
	total, err := s.users.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	resp := &userdto.UserListResponse{
		List:     make([]*userdto.UserListItem, 0, len(users)),
		Total:    uint64(total),
		Page:     page,
		PageSize: pageSize,
	}
	for _, u := range users {
		resp.List = append(resp.List, &userdto.UserListItem{
			ID:       u.ID,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
		})
	}
	return resp, nil
}

func (s *Service) GetAvatarUploadURL(ctx context.Context, userID uint64, fileExt string) (uploadURL, objectKey string, err error) {
	if s.avatars == nil {
		return "", "", common.ErrSystem
	}
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	if !s.allowedExts[ext] {
		return "", "", common.ErrInvalidRequestBody
	}

	objectKey = path.Join("avatar", fmt.Sprint(userID), uuid.NewString()+"."+ext)
	uploadURL, err = s.avatars.PresignedPutURL(ctx, objectKey, 10*time.Minute)
	if err != nil {
		return "", "", err
	}
	return uploadURL, objectKey, nil
}

func (s *Service) ConfirmAvatar(ctx context.Context, userID uint64, objectKey string) (string, error) {
	if s.avatars == nil {
		return "", common.ErrSystem
	}
	expectedPrefix := "avatar/" + fmt.Sprint(userID) + "/"
	if !strings.HasPrefix(objectKey, expectedPrefix) {
		return "", common.ErrInvalidRequestBody
	}

	user, err := s.findUser(ctx, userID)
	if err != nil {
		return "", err
	}
	user.Avatar = objectKey
	user.UpdatedTime = time.Now()
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return "", err
	}
	return s.avatars.GetObjectURL(s.publicDomain, objectKey), nil
}

func (s *Service) findUser(ctx context.Context, userID uint64) (*domainidentity.User, error) {
	user, err := s.users.FindUserByID(ctx, userID)
	if errors.Is(err, domainidentity.ErrUserNotFound) {
		return nil, common.ErrUserNotFound
	}
	return user, err
}

// 保留随机 token 生成能力，供后续兼容旧调用方使用。
func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
