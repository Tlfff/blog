package interceptor

import "context"

// Kind 身份类型：区分内部服务与外部合作方
type Kind string

const (
	// KindInternal 二方：内部服务调用
	KindInternal Kind = "internal"
	// KindExternal 三方：外部合作方调用
	KindExternal Kind = "external"
)

// Identity 认证通过后注入context的调用方身份。
// 对授权拦截器而言，它不关心ID具体叫什么，只关心"这个ID是否在方法的允许列表里"。
type Identity struct {
	Kind  Kind   // 身份类型，由服务端根据走通的认证路径赋值，客户端无法伪造
	ID    string // 凭证身份：internal=service_id，external=access_key_id
	Group string // 组织归属：internal=team_id，external=partner_id，仅用于统计/日志，不参与授权
}

type identityCtxKey struct{}
type identityHolderKey struct{}

// 共享身份容器：unary拦截器的ctx是值传递，
// 认证层写入的身份外层日志拦截器拿不到，通过指针holder跨层共享
type identityHolder struct {
	identity *Identity // 当前调用方身份
}

// 把身份注入context：同时写入值
// holder指针（供外层日志拦截器退出时读取）
func withIdentity(ctx context.Context, identity *Identity) context.Context {
	if h, ok := ctx.Value(identityHolderKey{}).(*identityHolder); ok {
		h.identity = identity
	}
	return context.WithValue(ctx, identityCtxKey{}, identity)
}

// IdentityFromContext 从 Context 中读取调用方身份。
func IdentityFromContext(ctx context.Context) (*Identity, bool) {
	id, ok := ctx.Value(identityCtxKey{}).(*Identity)
	return id, ok
}
