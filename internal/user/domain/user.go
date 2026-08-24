package domain

import "time"

const (
	RoleUser  int8 = 1 // 用户角色：普通用户
	RoleAdmin int8 = 2 // 用户角色：管理员

	StatusNormal  int8 = 1 // 用户状态：正常
	StatusDeleted int8 = 2 // 用户状态：已删除/禁用
)

// User 是 User 领域的聚合根。
type User struct {
	ID            uint64    // 用户唯一标识
	Nickname      string    // 用户昵称
	Phone         string    // 手机号
	Password      string    // PBKDF2加密后的密码：算法$迭代次数$Salt$Hash
	Avatar        string    // 用户头像URL
	Role          int8      // 用户角色：1-普通用户 2-管理员
	Status        int8      // 用户状态：1-正常 2-已删除/禁用
	LastLoginIP   string    // 上一次登录的IP地址
	LastLoginTime time.Time // 上一次登录的时间
	CreatedTime   time.Time // 创建时间
	UpdatedTime   time.Time // 最后更新时间
}

// 判断用户是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// 判断用户状态是否正常（未被删除/禁用）
func (u *User) IsNormal() bool {
	return u.Status == StatusNormal
}
