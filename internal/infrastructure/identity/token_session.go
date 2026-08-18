package identity

import (
	"blog/internal/auth"
	domainidentity "blog/internal/domain/identity"
	"context"
)

type tokenSessionAdapter struct {
	tokenAuth *auth.TokenAuth
}

// NewTokenSession 将现有 TokenAuth 适配为 Identity 领域会话 Port。
func NewTokenSession(tokenAuth *auth.TokenAuth) domainidentity.TokenSession {
	return &tokenSessionAdapter{tokenAuth: tokenAuth}
}

func (a *tokenSessionAdapter) CreateSession(ctx context.Context, userID uint64, role int8, ip, device string, rememberMe bool) (string, error) {
	return a.tokenAuth.CreateSession(ctx, userID, role, ip, device, rememberMe)
}

func (a *tokenSessionAdapter) GetSession(ctx context.Context, token string) (*domainidentity.Session, error) {
	session, err := a.tokenAuth.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return &domainidentity.Session{
		UserID:    session.UserID,
		Role:      session.Role,
		LoginTime: session.LoginTime,
		IP:        session.IP,
		Device:    session.Device,
	}, nil
}

func (a *tokenSessionAdapter) DeleteSession(ctx context.Context, token string) error {
	return a.tokenAuth.DeleteSession(ctx, token)
}

func (a *tokenSessionAdapter) DeleteAllSessions(ctx context.Context, userID uint64) error {
	return a.tokenAuth.DeleteAllSessions(ctx, userID)
}

func (a *tokenSessionAdapter) DeleteOtherSessions(ctx context.Context, userID uint64, currentToken string) error {
	return a.tokenAuth.DeleteOtherSessions(ctx, userID, currentToken)
}

func (a *tokenSessionAdapter) GetUserTokens(ctx context.Context, userID uint64) ([]string, error) {
	return a.tokenAuth.GetUserTokens(ctx, userID)
}
