package infrastructure

import (
	"blog/internal/platform/security"
	domainidentity "blog/internal/user/domain"
	"context"
)

// tokenSessionAdapter 是登录会话 Port 的 Redis Token 适配器。
type tokenSessionAdapter struct {
	tokenAuth *auth.TokenAuth // 底层 Redis 会话实现
}

// 将现有 TokenAuth 适配为 User 领域会话 Port
func NewTokenSession(tokenAuth *auth.TokenAuth) domainidentity.TokenSession {
	return &tokenSessionAdapter{tokenAuth: tokenAuth}
}

// 创建登录会话并返回 token
func (a *tokenSessionAdapter) CreateSession(ctx context.Context, userID uint64, role int8, ip, device string, rememberMe bool) (string, error) {
	return a.tokenAuth.CreateSession(ctx, userID, role, ip, device, rememberMe)
}

// 根据 token 查询会话，并转换为领域会话对象
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

// 删除指定 token 对应的会话，用于用户登出
func (a *tokenSessionAdapter) DeleteSession(ctx context.Context, token string) error {
	return a.tokenAuth.DeleteSession(ctx, token)
}

// 删除该用户的全部会话，用于强制全端下线
func (a *tokenSessionAdapter) DeleteAllSessions(ctx context.Context, userID uint64) error {
	return a.tokenAuth.DeleteAllSessions(ctx, userID)
}

// 删除该用户除当前 token 外的其他会话，用于改密后保留当前设备
func (a *tokenSessionAdapter) DeleteOtherSessions(ctx context.Context, userID uint64, currentToken string) error {
	return a.tokenAuth.DeleteOtherSessions(ctx, userID, currentToken)
}

// 查询该用户当前全部有效 token，用于后台管理
func (a *tokenSessionAdapter) GetUserTokens(ctx context.Context, userID uint64) ([]string, error) {
	return a.tokenAuth.GetUserTokens(ctx, userID)
}
