package infra

import (
	"blog/internal/platform/security"
	domainidentity "blog/internal/user/domain"
	"context"
)

// tokenSessionAdapter 是登录会话 Port 的 Redis Token 适配器。
type tokenSessionAdapter struct {
	tokenAuth *auth.TokenAuth // 底层 Redis 会话实现
}

// NewTokenSession 将 TokenAuth 适配为 User 会话 Port。
func NewTokenSession(tokenAuth *auth.TokenAuth) domainidentity.TokenSession {
	return &tokenSessionAdapter{tokenAuth: tokenAuth}
}

// CreateSession 创建登录会话并返回 Token。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - userID：会话所属用户唯一标识。
//   - role：用户角色：1-普通用户；2-管理员。
//   - ip：登录来源 IP。
//   - device：登录设备标识。
//   - rememberMe：是否延长 Token 有效期。
func (a *tokenSessionAdapter) CreateSession(ctx context.Context, userID uint64, role int8, ip, device string, rememberMe bool) (string, error) {
	return a.tokenAuth.CreateSession(ctx, userID, role, ip, device, rememberMe)
}

// GetSession 根据 Token 查询并转换领域会话对象。
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

// DeleteSession 删除指定 Token 会话。
func (a *tokenSessionAdapter) DeleteSession(ctx context.Context, token string) error {
	return a.tokenAuth.DeleteSession(ctx, token)
}

// DeleteAllSessions 删除用户全部会话。
func (a *tokenSessionAdapter) DeleteAllSessions(ctx context.Context, userID uint64) error {
	return a.tokenAuth.DeleteAllSessions(ctx, userID)
}

// DeleteOtherSessions 删除用户除当前 Token 外的其他会话。
func (a *tokenSessionAdapter) DeleteOtherSessions(ctx context.Context, userID uint64, currentToken string) error {
	return a.tokenAuth.DeleteOtherSessions(ctx, userID, currentToken)
}

// GetUserTokens 查询用户全部有效 Token。
func (a *tokenSessionAdapter) GetUserTokens(ctx context.Context, userID uint64) ([]string, error) {
	return a.tokenAuth.GetUserTokens(ctx, userID)
}
