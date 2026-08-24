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
	TokenPrefix      = "auth:token:"       // token 明文的 Redis Key 前缀，值为会话 JSON
	UserTokensPrefix = "auth:user-tokens:" // 用户 token 集合的 Redis Key 前缀，用于批量下线
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
// Session 是存入 Redis 的登录会话结构（值对象）。
type Session struct {
	UserID    uint64 `json:"user_id"`    // 会话所属用户ID
	Role      int8   `json:"role_id"`    // 登录时的用户角色：1-普通用户 2-管理员
	LoginTime int64  `json:"login_time"` // 登录时间（Unix 秒）
	IP        string `json:"ip"`         // 登录来源IP
	Device    string `json:"device"`     // 登录设备标识，如 web/ios/android
}

// Token 认证服务
// TokenAuth 是基于 Redis 的 Token 认证服务。
type TokenAuth struct {
	rdb SessionStore // Redis 会话存储，仅依赖窄接口便于测试替换
}

// 构造 Token 认证服务
func NewTokenAuth(rdb SessionStore) *TokenAuth {
	return &TokenAuth{rdb: rdb}
}

// 生成随机 token
// 生成 32 字节随机 token，并编码为十六进制字符串
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// 创建登录会话，返回 token
// 创建登录会话并返回 token
// rememberMe 为 true 时使用更长的有效期
func (t *TokenAuth) CreateSession(ctx context.Context, userID uint64, role int8, ip, device string, rememberMe bool) (string, error) {
	// 1. 生成随机 token
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	// 2. 按是否记住登录选择过期时间
	ttl := DefaultTokenTTL
	if rememberMe {
		ttl = RememberTokenTTL
	}

	// 3. 组装会话内容并序列化为 JSON
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
// 根据 token 查询登录会话，token 不存在或已过期时返回错误
func (t *TokenAuth) GetSession(ctx context.Context, token string) (*Session, error) {
	// 1. 从 Redis 读取会话 JSON
	data, err := t.rdb.Get(ctx, TokenPrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return nil, errors.New("token不存在或已过期")
	}
	if err != nil {
		return nil, err
	}

	// 2. 反序列化为会话结构
	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// 删除指定 token（登出）
// 删除指定 token，用于用户主动登出
func (t *TokenAuth) DeleteSession(ctx context.Context, token string) error {
	// 1. 先查询会话，拿到 user_id 用于清理集合
	// 先查询获取 user_id
	session, err := t.GetSession(ctx, token)
	if err != nil {
		return err
	}

	// 2. 删除 token 本体
	// 删除 token
	if err := t.rdb.Del(ctx, TokenPrefix+token).Err(); err != nil {
		return err
	}

	// 3. 从用户 token 集合中移除该 token
	// 从用户 token 集合中移除
	userTokensKey := UserTokensPrefix + fmt.Sprint(session.UserID)
	return t.rdb.SRem(ctx, userTokensKey, token).Err()
}

// 删除用户所有 token（强制下线）
// 删除用户全部 token，用于强制下线
func (t *TokenAuth) DeleteAllSessions(ctx context.Context, userID uint64) error {
	// 1. 拼接用户 token 集合 Key
	userTokensKey := UserTokensPrefix + fmt.Sprint(userID)

	// 2. 取出该用户全部 token
	// 获取所有 token
	tokens, err := t.rdb.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return err
	}

	// 3. 批量删除全部 token 本体
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

	// 4. 删除用户 token 集合
	// 删除集合
	return t.rdb.Del(ctx, userTokensKey).Err()
}

// 删除用户所有 token，除了当前使用的（修改密码后保留当前设备）
// 删除用户除当前 token 外的全部 token，用于改密后保留当前设备
func (t *TokenAuth) DeleteOtherSessions(ctx context.Context, userID uint64, currentToken string) error {
	userTokensKey := UserTokensPrefix + fmt.Sprint(userID)

	// 1. 取出该用户全部 token
	// 获取所有 token
	tokens, err := t.rdb.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return err
	}

	// 2. 逐个删除非当前 token
	// 删除除当前 token 外的所有 token
	for _, token := range tokens {
		if token != currentToken {
			if err := t.rdb.Del(ctx, TokenPrefix+token).Err(); err != nil {
				return err
			}
		}
	}

	// 3. 重建集合，仅保留当前 token
	// 重建集合，只保留当前 token
	if err := t.rdb.Del(ctx, userTokensKey).Err(); err != nil {
		return err
	}
	return t.rdb.SAdd(ctx, userTokensKey, currentToken).Err()
}

// 获取用户所有 token（用于后台管理）
// 获取用户全部 token，用于后台管理查看在线设备
func (t *TokenAuth) GetUserTokens(ctx context.Context, userID uint64) ([]string, error) {
	return t.rdb.SMembers(ctx, UserTokensPrefix+fmt.Sprint(userID)).Result()
}
