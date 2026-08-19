package infrastructure

import (
	"blog/internal/shared/common"
	domainidentity "blog/internal/user/domain"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 一次性改密凭证的 Redis Key 前缀与有效期
const (
	passwordChangeTokenPrefix = "user:password-change:" // 改密凭证Key前缀
	passwordChangeTokenTTL    = 10 * time.Minute        // 改密凭证有效期，10分钟
)

// passwordChangeTokenStore 是改密凭证 Port 的 Redis 适配器。
type passwordChangeTokenStore struct {
	rdb *redis.Client // Redis 客户端
}

// NewPasswordChangeTokenStore 提供一次性改密凭证的 Redis 适配器。
func NewPasswordChangeTokenStore(rdb *redis.Client) domainidentity.PasswordChangeTokenStore {
	return &passwordChangeTokenStore{rdb: rdb}
}

// 签发一次性改密凭证，校验旧密码通过后调用
func (s *passwordChangeTokenStore) Issue(ctx context.Context, userID uint64) (string, error) {
	if s.rdb == nil {
		return "", common.ErrSystem
	}
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", err
	}
	token := hex.EncodeToString(rawToken)
	if err := s.rdb.Set(ctx, passwordChangeTokenPrefix+token, userID, passwordChangeTokenTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// 消费一次性改密凭证并返回其绑定的用户ID，凭证用后即失效
func (s *passwordChangeTokenStore) Consume(ctx context.Context, token string) (uint64, error) {
	if s.rdb == nil {
		return 0, common.ErrSystem
	}
	value, err := s.rdb.GetDel(ctx, passwordChangeTokenPrefix+token).Result()
	if errors.Is(err, redis.Nil) {
		return 0, domainidentity.ErrPasswordChangeToken
	}
	if err != nil {
		return 0, err
	}
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, domainidentity.ErrPasswordChangeToken
	}
	return userID, nil
}
