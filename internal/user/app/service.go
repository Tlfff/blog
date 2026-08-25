// Package application 编排 User 上下文应用用例。
package app

import (
	iputil "blog/internal/platform/ip"
	apperrors "blog/internal/shared/apperrors"
	userdto "blog/internal/user/app/dto"
	domainidentity "blog/internal/user/domain"
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
	passwords      PasswordHasher                          // 密码哈希技术 Port
}

// NewService 创建 User Application 服务。
//
// 参数说明：
//   - users：用户持久化 Port。
//   - sessions：登录会话存储 Port。
//   - passwordTokens：一次性改密凭证存储 Port。
//   - avatars：头像对象存储 Port。
//   - passwords：密码哈希与校验 Port。
//   - publicDomain：对象存储公开访问域名。
//   - allowedExts：允许上传的头像扩展名列表。
func NewService(
	users domainidentity.UserRepository,
	sessions domainidentity.TokenSession,
	passwordTokens domainidentity.PasswordChangeTokenStore,
	avatars domainidentity.AvatarObjectStorage,
	passwords PasswordHasher,
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
		passwords:      passwords,
		publicDomain:   publicDomain,
		allowedExts:    extMap,
	}
}

// Register 注册新用户。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - phone：用户手机号。
//   - password：用户明文密码。
//   - nickname：用户昵称。
//   - clientIP：注册来源 IP。
func (s *Service) Register(ctx context.Context, phone, password, nickname, clientIP string) error {
	// 1. 查询账号是否已存在，未找到属于正常情况
	_, err := s.users.GetUserByAccount(ctx, phone, nickname)
	if err != nil && !errors.Is(err, domainidentity.ErrUserNotFound) {
		return err
	}

	// 2. 通过领域值对象校验新密码，再生成兼容哈希
	plainPassword, err := domainidentity.NewPlainPassword(password)
	if err != nil {
		return mapUserDomainError(err)
	}
	passwordHash, err := s.passwords.Hash(plainPassword)
	if err != nil {
		log.Printf("密码哈希失败: %v", err)
		return apperrors.ErrPasswordHashFailed
	}

	// 3. 通过领域构造函数创建用户并落库
	user := domainidentity.NewUser(nickname, phone, passwordHash, clientIP, time.Now())
	return s.users.CreateUser(ctx, user)
}

// Login 校验用户凭证并创建登录会话。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - phone：登录手机号，可以为空。
//   - nickname：登录昵称，可以为空。
//   - password：登录明文密码。
//   - clientIP：登录来源 IP。
//   - device：登录设备标识。
//   - rememberMe：是否延长会话有效期。
func (s *Service) Login(ctx context.Context, phone, nickname, password, clientIP, device string, rememberMe bool) (*userdto.LoginResponse, error) {
	// 1. 按手机号或昵称查询用户
	user, err := s.users.GetUserByAccount(ctx, phone, nickname)
	if err != nil {
		if errors.Is(err, domainidentity.ErrUserNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}

	// 2. 校验密码
	ok, err := s.passwords.Verify(password, user.Password)
	if err != nil {
		log.Printf("密码验证失败: %v", err)
		return nil, apperrors.ErrSystem
	}
	if !ok {
		return nil, apperrors.ErrPasswordFailed
	}

	// 3. 通过领域行为记录最后登录IP与时间，失败保存时仍只记录日志
	user.RecordLogin(clientIP, time.Now())
	if err := s.users.UpdateUser(ctx, user); err != nil {
		log.Printf("更新用户登录信息失败: %v", err)
	}

	// 4. 创建登录会话并返回访问令牌
	if s.sessions == nil {
		return nil, apperrors.ErrSystem
	}
	token, err := s.sessions.CreateSession(ctx, user.ID, user.Role, clientIP, device, rememberMe)
	if err != nil {
		log.Printf("创建登录会话失败: %v", err)
		return nil, apperrors.ErrSystem
	}
	return &userdto.LoginResponse{AccessToken: token}, nil
}

// Logout 删除当前用户会话。
func (s *Service) Logout(ctx context.Context, token string) error {
	if s.sessions == nil {
		return apperrors.ErrSystem
	}
	return s.sessions.DeleteSession(ctx, token)
}

// GetMyProfile 获取当前登录用户的完整资料。
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

// GetUserProfile 获取指定用户的公开资料。
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

// UpdateProfile 更新当前用户的昵称与头像。
func (s *Service) UpdateProfile(ctx context.Context, userID uint64, nickname, avatar string) error {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	user.UpdateProfile(nickname, avatar, time.Now())
	return s.users.UpdateUser(ctx, user)
}

// VerifyOldPassword 校验旧密码并签发一次性改密凭证。
func (s *Service) VerifyOldPassword(ctx context.Context, userID uint64, oldPassword string) (string, error) {
	// 1. 校验改密凭证存储是否可用
	if s.passwordTokens == nil {
		return "", apperrors.ErrSystem
	}
	// 2. 查询用户
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return "", err
	}
	// 3. 校验旧密码
	ok, err := s.passwords.Verify(oldPassword, user.Password)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", apperrors.ErrPasswordFailed
	}
	// 4. 签发一次性改密凭证
	return s.passwordTokens.Issue(ctx, userID)
}

// UpdatePassword 校验旧密码后修改密码，供内部调用方兼容使用。
func (s *Service) UpdatePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	// 1. 查询用户
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	// 2. 校验旧密码
	ok, err := s.passwords.Verify(oldPassword, user.Password)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.ErrPasswordFailed
	}
	// 3. 校验新密码、生成哈希并通过领域行为修改密码
	plainPassword, err := domainidentity.NewPlainPassword(newPassword)
	if err != nil {
		return mapUserDomainError(err)
	}
	hash, err := s.passwords.Hash(plainPassword)
	if err != nil {
		return err
	}
	user.ChangePassword(hash, time.Now())
	return s.users.UpdateUser(ctx, user)
}

// ChangePassword 使用一次性改密凭证修改密码，并删除其他设备会话。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - userID：当前用户唯一标识。
//   - changeToken：一次性密码修改凭证。
//   - newPassword：新明文密码。
//   - currentToken：当前设备登录 Token。
func (s *Service) ChangePassword(ctx context.Context, userID uint64, changeToken, newPassword, currentToken string) error {
	// 1. 校验改密凭证存储是否可用
	if s.passwordTokens == nil {
		return apperrors.ErrSystem
	}
	// 2. 消费一次性改密凭证，凭证只能使用一次
	tokenUserID, err := s.passwordTokens.Consume(ctx, changeToken)
	if errors.Is(err, domainidentity.ErrPasswordChangeToken) {
		return apperrors.ErrPasswordChangeToken
	}
	if err != nil {
		return err
	}
	// 3. 校验凭证归属，防止越权改他人密码
	if tokenUserID != userID {
		return apperrors.ErrPasswordChangeToken
	}

	// 4. 校验新密码并生成兼容哈希
	plainPassword, err := domainidentity.NewPlainPassword(newPassword)
	if err != nil {
		return mapUserDomainError(err)
	}
	hash, err := s.passwords.Hash(plainPassword)
	if err != nil {
		return err
	}
	// 5. 查询用户并通过领域行为写回新密码
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}
	user.ChangePassword(hash, time.Now())
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return err
	}
	// 6. 踢掉除当前设备外的所有会话
	if s.sessions != nil {
		return s.sessions.DeleteOtherSessions(ctx, userID, currentToken)
	}
	return nil
}

// UpdateAccount 变更账号绑定的手机号。
func (s *Service) UpdateAccount(ctx context.Context, userID uint64, phone string) error {
	// 1. 查询用户
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}

	// 2. 校验新手机号是否已被他人注册
	existing, err := s.users.GetUserByAccount(ctx, phone, "")
	if err == nil && existing.ID != userID {
		return apperrors.ErrPhoneAlreadyExists
	}
	if err != nil && !errors.Is(err, domainidentity.ErrUserNotFound) {
		return err
	}

	// 3. 通过领域行为写回新手机号
	user.ChangePhone(phone, time.Now())
	return s.users.UpdateUser(ctx, user)
}

// GetUserBasicInfo 获取用户基本信息，供二方服务调用。
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

// ListUserBasicInfo 分页获取用户基本信息列表。
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

// GetAvatarUploadURL 签发头像上传的预签名 URL。
func (s *Service) GetAvatarUploadURL(ctx context.Context, userID uint64, fileExt string) (uploadURL, objectKey string, err error) {
	// 1. 校验头像存储是否可用
	if s.avatars == nil {
		return "", "", apperrors.ErrSystem
	}
	// 2. 校验扩展名是否在白名单内
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	if !s.allowedExts[ext] {
		return "", "", apperrors.ErrInvalidRequestBody
	}

	// 3. 生成对象键并签发预签名上传URL
	objectKey = path.Join("avatar", fmt.Sprint(userID), uuid.NewString()+"."+ext)
	uploadURL, err = s.avatars.PresignedPutURL(ctx, objectKey, 10*time.Minute)
	if err != nil {
		return "", "", err
	}
	return uploadURL, objectKey, nil
}

// ConfirmAvatar 确认头像上传完成并写回用户资料。
func (s *Service) ConfirmAvatar(ctx context.Context, userID uint64, objectKey string) (string, error) {
	// 1. 校验头像存储是否可用
	if s.avatars == nil {
		return "", apperrors.ErrSystem
	}
	// 2. 校验对象键归属当前用户，防止越权覆盖他人头像
	expectedPrefix := "avatar/" + fmt.Sprint(userID) + "/"
	if !strings.HasPrefix(objectKey, expectedPrefix) {
		return "", apperrors.ErrInvalidRequestBody
	}

	// 3. 查询用户并写回头像
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return "", err
	}
	user.ChangeAvatar(objectKey, time.Now())
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return "", err
	}
	// 4. 返回头像的完整访问URL
	return s.avatars.GetObjectURL(s.publicDomain, objectKey), nil
}

// mapUserDomainError 将 User 领域错误映射为现有应用错误。
func mapUserDomainError(err error) error {
	switch {
	case errors.Is(err, domainidentity.ErrPasswordTooShort):
		return apperrors.ErrPasswordTooShort
	case errors.Is(err, domainidentity.ErrUserNotFound):
		return apperrors.ErrUserNotFound
	case errors.Is(err, domainidentity.ErrPasswordChangeToken):
		return apperrors.ErrPasswordChangeToken
	case errors.Is(err, domainidentity.ErrPhoneAlreadyExists):
		return apperrors.ErrPhoneAlreadyExists
	default:
		return err
	}
}

// findUser 按 ID 查询用户，并把领域错误映射为统一业务错误。
func (s *Service) findUser(ctx context.Context, userID uint64) (*domainidentity.User, error) {
	user, err := s.users.FindUserByID(ctx, userID)
	if errors.Is(err, domainidentity.ErrUserNotFound) {
		return nil, apperrors.ErrUserNotFound
	}
	return user, err
}

// randomToken 生成兼容旧调用方的随机 Token。
func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// GetUserSnapshot 返回跨上下文查询所需的最小用户快照。
//
// 返回值说明：
//   - id：用户唯一标识。
//   - nickname：用户昵称。
//   - avatar：用户头像地址。
//   - lastLoginIP：用户最后登录 IP。
//   - err：查询失败原因。
func (s *Service) GetUserSnapshot(ctx context.Context, userID uint64) (id uint64, nickname, avatar, lastLoginIP string, err error) {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return 0, "", "", "", err
	}
	return user.ID, user.Nickname, user.Avatar, user.LastLoginIP, nil
}

// BatchGetUserSnapshots 批量返回公开用户快照，避免列表组装产生 N+1 查询。
func (s *Service) BatchGetUserSnapshots(ctx context.Context, userIDs []uint64) (map[uint64]domainidentity.User, error) {
	users, err := s.users.FindUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uint64]domainidentity.User, len(users))
	for _, user := range users {
		result[user.ID] = *user
	}
	return result, nil
}
