package app

import (
	apperrors "blog/internal/shared/apperrors"
	domainidentity "blog/internal/user/domain"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// fakePasswordHasher 是 User Application 测试使用的密码哈希器。
type fakePasswordHasher struct{}

// Hash 生成可预测的测试密码哈希。
func (fakePasswordHasher) Hash(password domainidentity.PlainPassword) (domainidentity.PasswordHash, error) {
	return domainidentity.RestorePasswordHash("hash:" + password.String()), nil
}

// Verify 校验测试明文密码和哈希是否匹配。
func (fakePasswordHasher) Verify(password string, stored domainidentity.PasswordHash) (bool, error) {
	return stored.String() == "hash:"+password, nil
}

// fakeUserRepo 是 User Application 测试使用的内存 Repository。
type fakeUserRepo struct {
	users   map[uint64]*domainidentity.User // 用户唯一标识到用户聚合的映射
	nextID  uint64                          // 下一条用户记录的自增标识
	byPhone map[string]uint64               // 手机号到用户唯一标识的映射
	byNick  map[string]uint64               // 昵称到用户唯一标识的映射
}

// newFakeUserRepo 创建内存用户 Repository。
func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:   make(map[uint64]*domainidentity.User),
		nextID:  1,
		byPhone: make(map[string]uint64),
		byNick:  make(map[string]uint64),
	}
}

// CreateUser 保存用户并回填唯一标识。
func (f *fakeUserRepo) CreateUser(_ context.Context, user *domainidentity.User) error {
	if user.ID == 0 {
		user.ID = f.nextID
		f.nextID++
	}
	f.users[user.ID] = cloneUser(user)
	f.byPhone[user.Phone] = user.ID
	f.byNick[user.Nickname] = user.ID
	return nil
}

// GetUserByAccount 根据手机号或昵称查询用户。
func (f *fakeUserRepo) GetUserByAccount(_ context.Context, phone, nickname string) (*domainidentity.User, error) {
	if id, ok := f.byPhone[phone]; ok {
		return cloneUser(f.users[id]), nil
	}
	if id, ok := f.byNick[nickname]; ok {
		return cloneUser(f.users[id]), nil
	}
	return nil, domainidentity.ErrUserNotFound
}

// FindUserByID 根据唯一标识查询用户。
func (f *fakeUserRepo) FindUserByID(_ context.Context, id uint64) (*domainidentity.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, domainidentity.ErrUserNotFound
	}
	return cloneUser(user), nil
}

// FindUsersByIDs 批量查询用户。
func (f *fakeUserRepo) FindUsersByIDs(_ context.Context, ids []uint64) ([]*domainidentity.User, error) {
	users := make([]*domainidentity.User, 0, len(ids))
	for _, id := range ids {
		if user, ok := f.users[id]; ok {
			users = append(users, cloneUser(user))
		}
	}
	return users, nil
}

// UpdateUser 覆盖保存用户聚合。
func (f *fakeUserRepo) UpdateUser(_ context.Context, user *domainidentity.User) error {
	f.users[user.ID] = cloneUser(user)
	f.byPhone[user.Phone] = user.ID
	f.byNick[user.Nickname] = user.ID
	return nil
}

// GetUserList 分页查询用户列表。
func (f *fakeUserRepo) GetUserList(_ context.Context, page, pageSize int, _ bool) ([]*domainidentity.User, error) {
	ids := make([]uint64, 0, len(f.users))
	for id := range f.users {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	start := (page - 1) * pageSize
	if start >= len(ids) {
		return []*domainidentity.User{}, nil
	}
	end := start + pageSize
	if end > len(ids) {
		end = len(ids)
	}
	users := make([]*domainidentity.User, 0, end-start)
	for _, id := range ids[start:end] {
		users = append(users, cloneUser(f.users[id]))
	}
	return users, nil
}

// CountUsers 统计用户数量。
func (f *fakeUserRepo) CountUsers(_ context.Context) (int64, error) {
	return int64(len(f.users)), nil
}

// fakeTokenSession 是 User Application 测试使用的内存会话存储。
type fakeTokenSession struct {
	sessions     map[string]domainidentity.Session // Token 到会话的映射
	userSessions map[uint64][]string               // 用户唯一标识到 Token 列表的映射
	next         int                               // 下一个测试 Token 序号
}

// newFakeTokenSession 创建内存会话存储。
func newFakeTokenSession() *fakeTokenSession {
	return &fakeTokenSession{
		sessions:     make(map[string]domainidentity.Session),
		userSessions: make(map[uint64][]string),
	}
}

// CreateSession 创建测试登录会话。
//
// 参数说明：
//   - ctx：测试上下文，本实现不使用。
//   - userID：会话所属用户唯一标识。
//   - role：用户角色。
//   - ip：登录来源 IP。
//   - device：登录设备标识。
//   - rememberMe：是否延长会话有效期，本实现不使用。
func (f *fakeTokenSession) CreateSession(_ context.Context, userID uint64, role int8, ip, device string, _ bool) (string, error) {
	f.next++
	token := fmt.Sprintf("token-%d-%d", userID, f.next)
	f.sessions[token] = domainidentity.Session{
		UserID:    userID,
		Role:      role,
		LoginTime: time.Now().Unix(),
		IP:        ip,
		Device:    device,
	}
	f.userSessions[userID] = append(f.userSessions[userID], token)
	return token, nil
}

// GetSession 根据 Token 查询会话。
func (f *fakeTokenSession) GetSession(_ context.Context, token string) (*domainidentity.Session, error) {
	session, ok := f.sessions[token]
	if !ok {
		return nil, errors.New("token不存在或已过期")
	}
	return &session, nil
}

// DeleteSession 删除指定 Token 会话。
func (f *fakeTokenSession) DeleteSession(_ context.Context, token string) error {
	if session, ok := f.sessions[token]; ok {
		delete(f.sessions, token)
		tokens := f.userSessions[session.UserID]
		for i, t := range tokens {
			if t == token {
				f.userSessions[session.UserID] = append(tokens[:i], tokens[i+1:]...)
				break
			}
		}
	}
	return nil
}

// DeleteAllSessions 删除用户全部会话。
func (f *fakeTokenSession) DeleteAllSessions(_ context.Context, userID uint64) error {
	for _, token := range f.userSessions[userID] {
		delete(f.sessions, token)
	}
	delete(f.userSessions, userID)
	return nil
}

// DeleteOtherSessions 删除用户除当前 Token 外的其他会话。
func (f *fakeTokenSession) DeleteOtherSessions(_ context.Context, userID uint64, currentToken string) error {
	for _, token := range f.userSessions[userID] {
		if token != currentToken {
			delete(f.sessions, token)
		}
	}
	f.userSessions[userID] = []string{currentToken}
	return nil
}

// GetUserTokens 查询用户当前全部 Token。
func (f *fakeTokenSession) GetUserTokens(_ context.Context, userID uint64) ([]string, error) {
	return f.userSessions[userID], nil
}

// fakePasswordStore 是 User Application 测试使用的一次性改密凭证存储。
type fakePasswordStore struct {
	tokens map[string]uint64 // 凭证到用户唯一标识的映射
	next   int               // 下一个测试凭证序号
}

// newFakePasswordStore 创建内存改密凭证存储。
func newFakePasswordStore() *fakePasswordStore {
	return &fakePasswordStore{tokens: make(map[string]uint64)}
}

// Issue 签发测试改密凭证。
func (f *fakePasswordStore) Issue(_ context.Context, userID uint64) (string, error) {
	f.next++
	token := fmt.Sprintf("change-%d-%d", userID, f.next)
	f.tokens[token] = userID
	return token, nil
}

// Consume 消费测试改密凭证。
func (f *fakePasswordStore) Consume(_ context.Context, token string) (uint64, error) {
	userID, ok := f.tokens[token]
	if !ok {
		return 0, domainidentity.ErrPasswordChangeToken
	}
	delete(f.tokens, token)
	return userID, nil
}

// fakeAvatarStorage 是 User Application 测试使用的对象存储。
type fakeAvatarStorage struct{}

// PresignedPutURL 返回测试头像上传地址。
func (fakeAvatarStorage) PresignedPutURL(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	return "https://upload.example/" + objectKey, nil
}

// GetObjectURL 返回测试头像访问地址。
func (fakeAvatarStorage) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}

// DeleteObject 模拟删除头像对象。
func (fakeAvatarStorage) DeleteObject(_ context.Context, _ string) error {
	return nil
}

// newTestService 创建具备完整测试依赖的 User Application 服务。
func newTestService() *Service {
	return NewService(
		newFakeUserRepo(),
		newFakeTokenSession(),
		newFakePasswordStore(),
		fakeAvatarStorage{},
		fakePasswordHasher{},
		"https://cdn.example",
		[]string{"jpg", "png"},
	)
}

// TestService_RegisterLoginLogout 验证注册、登录和退出流程。
func TestService_RegisterLoginLogout(t *testing.T) {
	s := newTestService()
	ctx := context.Background()

	if err := s.Register(ctx, "13800000000", "123456", "测试用户", "127.0.0.1"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	resp, err := s.Login(ctx, "13800000000", "", "123456", "127.0.0.1", "web", false)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("登录响应缺少 access_token")
	}
	if err := s.Logout(ctx, resp.AccessToken); err != nil {
		t.Fatalf("退出失败: %v", err)
	}
}

// TestService_ChangePasswordInvalidatesOtherSessions 验证修改密码后其他设备会话失效。
func TestService_ChangePasswordInvalidatesOtherSessions(t *testing.T) {
	s := newTestService()
	ctx := context.Background()

	if err := s.Register(ctx, "13900000000", "oldpass", "改密用户", "127.0.0.1"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	login, err := s.Login(ctx, "13900000000", "", "oldpass", "127.0.0.1", "web", false)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	otherLogin, err := s.Login(ctx, "13900000000", "", "oldpass", "127.0.0.1", "ios", false)
	if err != nil {
		t.Fatalf("第二个会话登录失败: %v", err)
	}

	changeToken, err := s.VerifyOldPassword(ctx, 1, "oldpass")
	if err != nil {
		t.Fatalf("验证旧密码失败: %v", err)
	}
	if err := s.ChangePassword(ctx, 1, changeToken, "newpass", login.AccessToken); err != nil {
		t.Fatalf("改密失败: %v", err)
	}

	if _, err := s.Login(ctx, "13900000000", "", "oldpass", "127.0.0.1", "web", false); !errors.Is(err, apperrors.ErrPasswordFailed) {
		t.Fatalf("旧密码应登录失败, got %v", err)
	}
	if _, err := s.Login(ctx, "13900000000", "", "newpass", "127.0.0.1", "web", false); err != nil {
		t.Fatalf("新密码应登录成功: %v", err)
	}
	if _, err := s.sessions.GetSession(ctx, otherLogin.AccessToken); err == nil {
		t.Fatal("改密后其他设备会话应已失效")
	}
}

// TestService_AvatarContract 验证头像上传和确认契约。
func TestService_AvatarContract(t *testing.T) {
	s := newTestService()
	ctx := context.Background()

	if err := s.Register(ctx, "13700000000", "123456", "头像用户", "127.0.0.1"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	if _, _, err := s.GetAvatarUploadURL(ctx, 1, "exe"); !errors.Is(err, apperrors.ErrInvalidRequestBody) {
		t.Fatalf("非法扩展名应返回参数错误, got %v", err)
	}
	_, objectKey, err := s.GetAvatarUploadURL(ctx, 1, "png")
	if err != nil {
		t.Fatalf("获取上传凭证失败: %v", err)
	}
	if !strings.HasPrefix(objectKey, "avatar/1/") {
		t.Fatalf("object key 前缀不符合归属规则: %s", objectKey)
	}
	if _, err := s.ConfirmAvatar(ctx, 1, "avatar/2/hack.png"); !errors.Is(err, apperrors.ErrInvalidRequestBody) {
		t.Fatalf("跨用户头像 key 应被拒绝, got %v", err)
	}
	url, err := s.ConfirmAvatar(ctx, 1, objectKey)
	if err != nil {
		t.Fatalf("确认头像失败: %v", err)
	}
	if url != "https://cdn.example/"+objectKey {
		t.Fatalf("头像 URL 契约被改变: %s", url)
	}
}

// TestService_AdminRole 验证管理员角色保持不变。
func TestService_AdminRole(t *testing.T) {
	repo := newFakeUserRepo()
	s := NewService(repo, nil, nil, nil, fakePasswordHasher{}, "", nil)
	ctx := context.Background()
	if err := repo.CreateUser(ctx, &domainidentity.User{
		ID:       1,
		Nickname: "管理员",
		Phone:    "13600000000",
		Password: domainidentity.RestorePasswordHash("hash"),
		Role:     domainidentity.RoleAdmin,
		Status:   domainidentity.StatusNormal,
	}); err != nil {
		t.Fatalf("创建管理员失败: %v", err)
	}
	profile, err := s.GetMyProfile(ctx, 1)
	if err != nil {
		t.Fatalf("获取资料失败: %v", err)
	}
	if profile.Role != domainidentity.RoleAdmin {
		t.Fatalf("管理员角色被改变: %d", profile.Role)
	}
}

// cloneUser 复制用户聚合，避免测试数据被调用方直接修改。
func cloneUser(u *domainidentity.User) *domainidentity.User {
	if u == nil {
		return nil
	}
	clone := *u
	return &clone
}
