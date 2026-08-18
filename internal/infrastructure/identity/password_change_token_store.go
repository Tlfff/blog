package identity

import (
	"blog/internal/common"
	domainidentity "blog/internal/domain/identity"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	passwordChangeTokenPrefix = "user:password-change:"
	passwordChangeTokenTTL    = 10 * time.Minute
)

type passwordChangeTokenStore struct {
	rdb *redis.Client
}

// NewPasswordChangeTokenStore 提供一次性改密凭证的 Redis 适配器。
func NewPasswordChangeTokenStore(rdb *redis.Client) domainidentity.PasswordChangeTokenStore {
	return &passwordChangeTokenStore{rdb: rdb}
}

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
