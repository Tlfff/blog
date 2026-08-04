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

// 从context中取出调用方身份（认证拦截器注入，授权/业务层使用）
func IdentityFromContext(ctx context.Context) (*Identity, bool) {
	id, ok := ctx.Value(identityCtxKey{}).(*Identity)
	return id, ok
}
