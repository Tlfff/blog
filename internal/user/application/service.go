// Package application 编排 User 上下文应用用例。
package application

import (
	"blog/internal/shared/common"
	userdto "blog/internal/user/application/dto"
	domainidentity "blog/internal/user/domain"
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

// Service 编排注册、登录、会话、资料与头像用例。
type Service struct {
	users          domainidentity.UserRepository           // 用户持久化 Port
	sessions       domainidentity.TokenSession             // 登录会话存储 Port
	passwordTokens domainidentity.PasswordChangeTokenStore // 一次性改密凭证存储 Port
	avatars        domainidentity.AvatarObjectStorage      // 头像对象存储 Port
	publicDomain   string                                  // 对象存储对外访问域名
	allowedExts    map[string]bool                         // 允许上传的头像扩展名集合
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

// 注册新用户
func (s *Service) Register(ctx context.Context, phone, password, nickname, clientIP string) error {
	// 1. 查询账号是否已存在，未找到属于正常情况
	_, err := s.users.GetUserByAccount(ctx, phone, nickname)
	if err != nil && !errors.Is(err, domainidentity.ErrUserNotFound) {
		return err
	}

	// 2. 生成密码哈希
	passwordHash, err := domainidentity.HashPassword(password)
	if err != nil {
		log.Printf("密码哈希失败: %v", err)
		return common.ErrPasswordHashFailed
	}

	// 3. 组装用户并落库，默认普通用户与正常状态
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

// 用户登录并创建会话
func (s *Service) Login(ctx context.Context, phone, nickname, password, clientIP, device string, rememberMe bool) (*userdto.LoginResponse, error) {
	// 1. 按手机号或昵称查询用户
	user, err := s.users.GetUserByAccount(ctx, phone, nickname)
	if err != nil {
		if errors.Is(err, domainidentity.ErrUserNotFound) {
			return nil, common.ErrUserNotFound
		}
		return nil, err
	}

	// 2. 校验密码
	ok, err := domainidentity.VerifyPassword(password, user.Password)
	if err != nil {
		log.Printf("密码验证失败: %v", err)
		return nil, common.ErrSystem
	}
	if !ok {
		return nil, common.ErrPasswordFailed
	}

	// 3. 更新最后登录IP与时间，失败只记录日志不阻断登录
	user.LastLoginIP = clientIP
	user.LastLoginTime = time.Now()
	user.UpdatedTime = time.Now()
	if err := s.users.UpdateUser(ctx, user); err != nil {
		log.Printf("更新用户登录信息失败: %v", err)
	}

	// 4. 创建登录会话并返回访问令牌
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

// 用户登出，删除当前会话
func (s *Service) Logout(ctx context.Context, token string) error {
	if s.sessions == nil {
		return common.ErrSystem
	}
	return s.sessions.DeleteSession(ctx, token)
}

// 获取当前登录用户的完整资料
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

// 获取他人主页的公开资料
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

// 更新当前用户的昵称与头像
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

// 校验旧密码并签发一次性改密凭证
func (s *Service) VerifyOldPassword(ctx context.Context, userID uint64, oldPassword string) (string, error) {
	// 1. 校验改密凭证存储是否可用
	if s.passwordTokens == nil {
		return "", common.ErrSystem
	}
	// 2. 查询用户
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return "", err
	}
	// 3. 校验旧密码
	ok, err := domainidentity.VerifyPassword(oldPassword, user.Password)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", common.ErrPasswordFailed
	}
	// 4. 签发一次性改密凭证
	return s.passwordTokens.Issue(ctx, userID)
}

// 直接用旧密码校验后修改密码，供内部调用方兼容使用
func (s *Service) UpdatePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	// 1. 查询用户
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	// 2. 校验旧密码
	ok, err := domainidentity.VerifyPassword(oldPassword, user.Password)
	if err != nil {
		return err
	}
	if !ok {
		return common.ErrPasswordFailed
	}
	// 3. 生成新密码哈希并落库
	hash, err := domainidentity.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.Password = hash
	user.UpdatedTime = time.Now()
	return s.users.UpdateUser(ctx, user)
}

// 凭一次性改密凭证修改密码，并踢掉其他设备的会话
func (s *Service) ChangePassword(ctx context.Context, userID uint64, changeToken, newPassword, currentToken string) error {
	// 1. 校验改密凭证存储是否可用
	if s.passwordTokens == nil {
		return common.ErrSystem
	}
	// 2. 消费一次性改密凭证，凭证只能使用一次
	tokenUserID, err := s.passwordTokens.Consume(ctx, changeToken)
	if errors.Is(err, domainidentity.ErrPasswordChangeToken) {
		return common.ErrPasswordChangeToken
	}
	if err != nil {
		return err
	}
	// 3. 校验凭证归属，防止越权改他人密码
	if tokenUserID != userID {
		return common.ErrPasswordChangeToken
	}

	// 4. 生成新密码哈希
	hash, err := domainidentity.HashPassword(newPassword)
	if err != nil {
		return err
	}
	// 5. 查询用户并写回新密码
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	user.Password = hash
	user.UpdatedTime = time.Now()
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return err
	}
	// 6. 踢掉除当前设备外的所有会话
	if s.sessions != nil {
		return s.sessions.DeleteOtherSessions(ctx, userID, currentToken)
	}
	return nil
}

// 变更账号绑定的手机号
func (s *Service) UpdateAccount(ctx context.Context, userID uint64, phone string) error {
	// 1. 查询用户
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}

	// 2. 校验新手机号是否已被他人注册
	existing, err := s.users.GetUserByAccount(ctx, phone, "")
	if err == nil && existing.ID != userID {
		return common.ErrPhoneAlreadyExists
	}
	if err != nil && !errors.Is(err, domainidentity.ErrUserNotFound) {
		return err
	}

	// 3. 写回新手机号
	user.Phone = phone
	user.UpdatedTime = time.Now()
	return s.users.UpdateUser(ctx, user)
}

// 获取用户基本信息，供二方服务调用
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

// 分页获取用户基本信息列表
func (s *Service) ListUserBasicInfo(ctx context.Context, page, pageSize uint64, isDesc bool) (*userdto.UserListResponse, error) {
	// 1. 分页查询用户列表
	users, err := s.users.GetUserList(ctx, int(page), int(pageSize), isDesc)
	if err != nil {
		return nil, err
	}
	// 2. 统计用户总数
	total, err := s.users.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	// 3. 组装分页响应
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

// 签发头像上传的预签名URL
func (s *Service) GetAvatarUploadURL(ctx context.Context, userID uint64, fileExt string) (uploadURL, objectKey string, err error) {
	// 1. 校验头像存储是否可用
	if s.avatars == nil {
		return "", "", common.ErrSystem
	}
	// 2. 校验扩展名是否在白名单内
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	if !s.allowedExts[ext] {
		return "", "", common.ErrInvalidRequestBody
	}

	// 3. 生成对象键并签发预签名上传URL
	objectKey = path.Join("avatar", fmt.Sprint(userID), uuid.NewString()+"."+ext)
	uploadURL, err = s.avatars.PresignedPutURL(ctx, objectKey, 10*time.Minute)
	if err != nil {
		return "", "", err
	}
	return uploadURL, objectKey, nil
}

// 确认头像上传完成并写回用户资料
func (s *Service) ConfirmAvatar(ctx context.Context, userID uint64, objectKey string) (string, error) {
	// 1. 校验头像存储是否可用
	if s.avatars == nil {
		return "", common.ErrSystem
	}
	// 2. 校验对象键归属当前用户，防止越权覆盖他人头像
	expectedPrefix := "avatar/" + fmt.Sprint(userID) + "/"
	if !strings.HasPrefix(objectKey, expectedPrefix) {
		return "", common.ErrInvalidRequestBody
	}

	// 3. 查询用户并写回头像
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return "", err
	}
	user.Avatar = objectKey
	user.UpdatedTime = time.Now()
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return "", err
	}
	// 4. 返回头像的完整访问URL
	return s.avatars.GetObjectURL(s.publicDomain, objectKey), nil
}

// 按ID查询用户，并把领域错误映射为统一业务错误
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
