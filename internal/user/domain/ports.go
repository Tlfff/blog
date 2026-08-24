package domain

import (
	"context"
	"time"
)

// UserRepository 是 Identity 模块的用户持久化 Port。
type UserRepository interface {
	// 新建用户记录
	CreateUser(ctx context.Context, user *User) error
	// 按手机号或昵称查询用户，用于登录时的账号识别
	GetUserByAccount(ctx context.Context, phone, nickname string) (*User, error)
	// 按用户ID查询单个用户
	FindUserByID(ctx context.Context, id uint64) (*User, error)
	// 按用户ID批量查询用户，用于列表场景避免 N+1 查询
	FindUsersByIDs(ctx context.Context, ids []uint64) ([]*User, error)
	// 更新用户信息
	UpdateUser(ctx context.Context, user *User) error
	// 分页查询用户列表，isDesc 为 true 时按创建时间降序
	GetUserList(ctx context.Context, page, pageSize int, isDesc bool) ([]*User, error)
	// 统计用户总数，用于分页返回 total
	CountUsers(ctx context.Context) (int64, error)
}

// TokenSession 是登录会话存储 Port。
type TokenSession interface {
	// 创建登录会话并返回访问令牌，rememberMe 为 true 时延长有效期
	CreateSession(ctx context.Context, userID uint64, role int8, ip, device string, rememberMe bool) (string, error)
	// 按令牌读取会话信息，令牌无效或过期时返回错误
	GetSession(ctx context.Context, token string) (*Session, error)
	// 删除单个会话，用于登出
	DeleteSession(ctx context.Context, token string) error
	// 删除用户的全部会话，用于改密后强制全端下线
	DeleteAllSessions(ctx context.Context, userID uint64) error
	// 删除用户除当前令牌外的其他会话，用于改密后保留当前端登录态
	DeleteOtherSessions(ctx context.Context, userID uint64, currentToken string) error
	// 查询用户当前全部有效令牌
	GetUserTokens(ctx context.Context, userID uint64) ([]string, error)
}

// PasswordChangeTokenStore 是一次性改密凭证存储 Port。
type PasswordChangeTokenStore interface {
	// 校验旧密码通过后签发一次性改密凭证
	Issue(ctx context.Context, userID uint64) (string, error)
	// 消费改密凭证并返回其所属用户ID，凭证使用后立即失效
	Consume(ctx context.Context, token string) (uint64, error)
}

// AvatarObjectStorage 是头像对象存储 Port。
type AvatarObjectStorage interface {
	// 生成头像上传的预签名 PUT 地址，ttl 为地址有效期
	PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	// 拼装对象的公开访问地址
	GetObjectURL(publicDomain, objectKey string) string
	// 删除对象，用于替换头像后清理旧文件
	DeleteObject(ctx context.Context, objectKey string) error
}
