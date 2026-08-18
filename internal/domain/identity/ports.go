package identity

import (
	"context"
	"time"
)

// UserRepository 是 Identity 模块的用户持久化 Port。
type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByAccount(ctx context.Context, phone, nickname string) (*User, error)
	FindUserByID(ctx context.Context, id uint64) (*User, error)
	FindUsersByIDs(ctx context.Context, ids []uint64) ([]*User, error)
	UpdateUser(ctx context.Context, user *User) error
	GetUserList(ctx context.Context, page, pageSize int, isDesc bool) ([]*User, error)
	CountUsers(ctx context.Context) (int64, error)
}

// TokenSession 是登录会话存储 Port。
type TokenSession interface {
	CreateSession(ctx context.Context, userID uint64, role int8, ip, device string, rememberMe bool) (string, error)
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteAllSessions(ctx context.Context, userID uint64) error
	DeleteOtherSessions(ctx context.Context, userID uint64, currentToken string) error
	GetUserTokens(ctx context.Context, userID uint64) ([]string, error)
}

// PasswordChangeTokenStore 是一次性改密凭证存储 Port。
type PasswordChangeTokenStore interface {
	Issue(ctx context.Context, userID uint64) (string, error)
	Consume(ctx context.Context, token string) (uint64, error)
}

// AvatarObjectStorage 是头像对象存储 Port。
type AvatarObjectStorage interface {
	PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	GetObjectURL(publicDomain, objectKey string) string
	DeleteObject(ctx context.Context, objectKey string) error
}
