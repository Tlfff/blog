// Package auth 提供登录会话、Token、HMAC 签名等认证能力。
package auth

type userContextKey struct{} //空类型，可以让每个包的key都是独立类型

var userCtxKey = userContextKey{} //创建全局唯一的变量

// UserContext 是登录用户的请求上下文信息（值对象），由认证中间件写入 Gin 上下文。
type UserContext struct {
	UserID uint64 // 用户唯一标识
	Phone  string // 手机号
	Role   int8   // 用户角色：1-普通用户 2-管理员
}

// // 存
// func SetUserContext(ctx context.Context, user *UserContext) context.Context {
// 	return context.WithValue(ctx, userCtxKey, user)
// }

// // 取
// func GetUserContext(ctx context.Context) (*UserContext, bool) {
// 	user, ok := ctx.Value(userCtxKey).(*UserContext)
// 	return user, ok
// }
