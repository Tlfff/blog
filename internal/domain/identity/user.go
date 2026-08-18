package identity

import "time"

const (
	RoleUser  int8 = 1
	RoleAdmin int8 = 2

	StatusNormal  int8 = 1
	StatusDeleted int8 = 2
)

// User 是 Identity 领域的聚合根。
type User struct {
	ID            uint64
	Nickname      string
	Phone         string
	Password      string
	Avatar        string
	Role          int8
	Status        int8
	LastLoginIP   string
	LastLoginTime time.Time
	CreatedTime   time.Time
	UpdatedTime   time.Time
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsNormal() bool {
	return u.Status == StatusNormal
}
