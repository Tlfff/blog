package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func checkToken(c *gin.Context, rdb *redis.Client) (*auth.UserContext, string, error) {
	// 1. 校验Authorization请求头
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, "", common.ErrAuthorizationRequired
	}
	// 2. Bearer 格式校验
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, "", common.ErrInvalidAuthorizationHeader
	}
	// 3. 截取Token字符串
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, "", common.ErrTokenEmpty
	}
	// 4. 查询 Redis token 会话
	tokenAuth := auth.NewTokenAuth(rdb)
	session, err := tokenAuth.GetSession(c, token)
	if err != nil {
		return nil, "", common.ErrTokenInvalid
	}
	// 5. 组装用户信息并注入Gin上下文
	userCtx := &auth.UserContext{
		UserID: session.UserID,
		Role:   session.Role,
	}
	return userCtx, token, nil
}

// 必须登录的场景
func MustAuth(rdbs ...*redis.Client) gin.HandlerFunc {
	var rdb *redis.Client
	if len(rdbs) > 0 {
		rdb = rdbs[0]
	}
	return func(c *gin.Context) {
		userCtx, token, err := checkToken(c, rdb)
		// 未登录直接拦截
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		c.Set("currentUser", userCtx)
		c.Set("currentToken", token)
		c.Next()
	}
}

// 可选认证中间件，登录和未登录有不同逻辑
func OptionalAuth(rdbs ...*redis.Client) gin.HandlerFunc {
	var rdb *redis.Client
	if len(rdbs) > 0 {
		rdb = rdbs[0]
	}
	return func(c *gin.Context) {
		userCtx, token, err := checkToken(c, rdb)
		// 如果登录了就记录用户信息
		if err == nil {
			c.Set("currentUser", userCtx)
			c.Set("currentToken", token)
		}
		c.Next()
	}
}
