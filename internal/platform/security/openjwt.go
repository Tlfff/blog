package auth

import (
	apperrors "blog/internal/shared/apperrors"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	OpenJWTIssuer     = "blog-open"    // 二方服务JWT的签发者，与C端(blog)区分
	OpenJWTExpireTime = time.Hour * 24 // 二方token有效期
)

// 二方服务JWT签名密钥，由启动流程从配置文件注入，与C端用户JWT密钥完全隔离
var openJWTSecret []byte

// 注入二方服务JWT密钥，gRPC服务启动时调用一次
func InitOpenJWT(secret string) {
	openJWTSecret = []byte(secret)
}

// 二方服务token的Payload：
// service_id 标识调用方服务（授权主体），team_id 标识所属团队（仅用于统计，不参与授权）
type OpenClaims struct {
	ServiceID string `json:"service_id"` // 调用方服务标识，作为授权主体
	TeamID    string `json:"team_id"`    // 所属团队标识，仅用于统计，不参与授权
	jwt.RegisteredClaims
}

// 给内部服务颁发token（平台组统一下发）
// 给二方服务颁发 token，由平台组统一下发
// serviceID 为授权主体，teamID 仅用于统计
func OpenGenerateToken(serviceID, teamID string) (string, error) {
	// 1. 校验密钥是否已注入
	if len(openJWTSecret) == 0 {
		return "", errors.New("二方jwt密钥未初始化")
	}
	// 2. 组装 Claims，设置签发者与有效期
	now := time.Now()
	claims := &OpenClaims{
		ServiceID: serviceID,
		TeamID:    teamID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    OpenJWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(OpenJWTExpireTime)),
		},
	}
	// 3. 使用 HS256 签名并返回 token 字符串
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(openJWTSecret)
}

// 解析并校验二方token（gRPC拦截器使用）
// 解析并校验二方 token，供 gRPC 拦截器使用
func OpenVerifyToken(tokenString string) (*OpenClaims, error) {
	// 1. 解析 token 并校验签名、签发者与过期时间
	claims := &OpenClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if len(openJWTSecret) == 0 {
			return nil, errors.New("二方jwt密钥未初始化")
		}
		return openJWTSecret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // 锁定签名算法，防止算法混淆攻击
		jwt.WithIssuer(OpenJWTIssuer),                                // 校验签发者
		jwt.WithExpirationRequired(),                                 // exp必须存在且未过期
	)
	// 2. 将 jwt 库错误映射为统一业务错误
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, apperrors.ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, apperrors.ErrTokenSignature
		case errors.Is(err, jwt.ErrTokenInvalidIssuer):
			return nil, apperrors.ErrTokenIssuer
		default:
			return nil, apperrors.ErrTokenInvalid
		}
	}
	// 3. 校验通过，返回 Claims
	return claims, nil
}
