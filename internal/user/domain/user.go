package domain

import "time"

const (
	RoleUser  int8 = 1 // 用户角色：普通用户
	RoleAdmin int8 = 2 // 用户角色：管理员

	StatusNormal  int8 = 1 // 用户状态：正常
	StatusDeleted int8 = 2 // 用户状态：已删除/禁用

	DefaultAvatar = "https://example.com/default-avatar.png" // 新注册用户默认头像
)

// User 是 User 领域的聚合根。
type User struct {
	ID            uint64       // 用户唯一标识
	Nickname      string       // 用户昵称
	Phone         string       // 手机号
	Password      PasswordHash // PBKDF2 密码哈希
	Avatar        string       // 用户头像URL或对象 Key
	Role          int8         // 用户角色：1-普通用户 2-管理员
	Status        int8         // 用户状态：1-正常 2-已删除/禁用
	LastLoginIP   string       // 上一次登录的IP地址
	LastLoginTime time.Time    // 上一次登录的时间
	CreatedTime   time.Time    // 创建时间
	UpdatedTime   time.Time    // 最后更新时间
}

// NewUser 创建符合现有默认规则的用户聚合。
//
// 参数说明：
//   - nickname：用户昵称。
//   - phone：用户手机号。
//   - password：用户密码哈希。
//   - clientIP：注册来源 IP。
//   - now：用户创建时间。
func NewUser(nickname, phone string, password PasswordHash, clientIP string, now time.Time) *User {
	return &User{
		Nickname:      nickname,
		Phone:         phone,
		Password:      password,
		Avatar:        DefaultAvatar,
		Role:          RoleUser,
		Status:        StatusNormal,
		LastLoginIP:   clientIP,
		LastLoginTime: now,
		CreatedTime:   now,
		UpdatedTime:   now,
	}
}

// IsAdmin 判断用户是否为管理员。
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsNormal 判断用户状态是否正常。
func (u *User) IsNormal() bool {
	return u.Status == StatusNormal
}

// RecordLogin 记录本次成功登录信息。
func (u *User) RecordLogin(ip string, now time.Time) {
	u.LastLoginIP = ip
	u.LastLoginTime = now
	u.UpdatedTime = now
}

// UpdateProfile 更新用户昵称和头像资料。
func (u *User) UpdateProfile(nickname, avatar string, now time.Time) {
	u.Nickname = nickname
	u.Avatar = avatar
	u.UpdatedTime = now
}

// ChangePassword 替换用户密码哈希。
func (u *User) ChangePassword(password PasswordHash, now time.Time) {
	u.Password = password
	u.UpdatedTime = now
}

// ChangePhone 修改用户绑定手机号。
func (u *User) ChangePhone(phone string, now time.Time) {
	u.Phone = phone
	u.UpdatedTime = now
}

// ChangeAvatar 修改用户头像对象 Key。
func (u *User) ChangeAvatar(objectKey string, now time.Time) {
	u.Avatar = objectKey
	u.UpdatedTime = now
}
