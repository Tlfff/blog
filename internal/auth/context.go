package auth

type userContextKey struct{} //空类型，可以让每个包的key都是独立类型

var userCtxKey = userContextKey{} //创建全局唯一的变量

type UserContext struct {
	UserID uint64
	Phone  string
	Role   int8
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
