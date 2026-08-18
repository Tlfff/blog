package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	TokenPrefix      = "auth:token:"
	UserTokensPrefix = "auth:user-tokens:"
	DefaultTokenTTL  = 7 * 24 * time.Hour  // 7天
	RememberTokenTTL = 30 * 24 * time.Hour // 30天
)

// SessionStore 是 TokenAuth 需要的 Redis 会话存储能力。
// 使用窄接口而不是 *redis.Client，便于测试替换为内存实现。
type SessionStore interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	SAdd(ctx context.Context, key string, members ...any) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	SRem(ctx context.Context, key string, members ...any) *redis.IntCmd
}

// 登录会话信息
type Session struct {
	UserID    uint64 `json:"user_id"`
	Role      int8   `json:"role_id"`
	LoginTime int64  `json:"login_time"`
	IP        string `json:"ip"`
	Device    string `json:"device"`
}

// Token 认证服务
type TokenAuth struct {
	rdb SessionStore
}

func NewTokenAuth(rdb SessionStore) *TokenAuth {
	return &TokenAuth{rdb: rdb}
}

// 生成随机 token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// 创建登录会话，返回 token
func (t *TokenAuth) CreateSession(ctx context.Context, userID uint64, role int8, ip, device string, rememberMe bool) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	ttl := DefaultTokenTTL
	if rememberMe {
		ttl = RememberTokenTTL
	}

	session := &Session{
		UserID:    userID,
		Role:      role,
		LoginTime: time.Now().Unix(),
		IP:        ip,
		Device:    device,
	}

	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	// 1. 写入 token → session
	tokenKey := TokenPrefix + token
	if err := t.rdb.Set(ctx, tokenKey, data, ttl).Err(); err != nil {
		return "", err
	}

	// 2. 写入用户 token 集合
	userTokensKey := UserTokensPrefix + fmt.Sprint(userID)
	if err := t.rdb.SAdd(ctx, userTokensKey, token).Err(); err != nil {
		// 回滚 token
		t.rdb.Del(ctx, tokenKey)
		return "", err
	}

	return token, nil
}

// 根据 token 查询会话
func (t *TokenAuth) GetSession(ctx context.Context, token string) (*Session, error) {
	data, err := t.rdb.Get(ctx, TokenPrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return nil, errors.New("token不存在或已过期")
	}
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// 删除指定 token（登出）
func (t *TokenAuth) DeleteSession(ctx context.Context, token string) error {
	// 先查询获取 user_id
	session, err := t.GetSession(ctx, token)
	if err != nil {
		return err
	}

	// 删除 token
	if err := t.rdb.Del(ctx, TokenPrefix+token).Err(); err != nil {
		return err
	}

	// 从用户 token 集合中移除
	userTokensKey := UserTokensPrefix + fmt.Sprint(session.UserID)
	return t.rdb.SRem(ctx, userTokensKey, token).Err()
}

// 删除用户所有 token（强制下线）
func (t *TokenAuth) DeleteAllSessions(ctx context.Context, userID uint64) error {
	userTokensKey := UserTokensPrefix + fmt.Sprint(userID)

	// 获取所有 token
	tokens, err := t.rdb.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return err
	}

	// 删除所有 token
	if len(tokens) > 0 {
		keys := make([]string, len(tokens))
		for i, token := range tokens {
			keys[i] = TokenPrefix + token
		}
		if err := t.rdb.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}

	// 删除集合
	return t.rdb.Del(ctx, userTokensKey).Err()
}

// 删除用户所有 token，除了当前使用的（修改密码后保留当前设备）
func (t *TokenAuth) DeleteOtherSessions(ctx context.Context, userID uint64, currentToken string) error {
	userTokensKey := UserTokensPrefix + fmt.Sprint(userID)

	// 获取所有 token
	tokens, err := t.rdb.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return err
	}

	// 删除除当前 token 外的所有 token
	for _, token := range tokens {
		if token != currentToken {
			if err := t.rdb.Del(ctx, TokenPrefix+token).Err(); err != nil {
				return err
			}
		}
	}

	// 重建集合，只保留当前 token
	if err := t.rdb.Del(ctx, userTokensKey).Err(); err != nil {
		return err
	}
	return t.rdb.SAdd(ctx, userTokensKey, currentToken).Err()
}

// 获取用户所有 token（用于后台管理）
func (t *TokenAuth) GetUserTokens(ctx context.Context, userID uint64) ([]string, error) {
	return t.rdb.SMembers(ctx, UserTokensPrefix+fmt.Sprint(userID)).Result()
}
