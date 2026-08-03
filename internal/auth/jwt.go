package auth

import (
	"blog/internal/common"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	JWTIssuer     = "blog"         //JWT的签发者
	JWTExpireTime = time.Hour * 24 //JWT的过期时间，这里设置为24小时
)

// JWT签名密钥，由启动流程从配置文件注入，禁止硬编码在代码中
var jwtSecret []byte

// InitJWT 注入JWT签名密钥，服务启动时调用一次
func InitJWT(secret string) {
	jwtSecret = []byte(secret)
}

// 生成Token
func GenerateToken(phone string, role int8, userID uint64) (string, error) {
	if len(jwtSecret) == 0 {
		return "", errors.New("jwt密钥未初始化")
	}
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Phone:  phone,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(JWTExpireTime)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

// 解析和验证Token
func VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if len(jwtSecret) == 0 {
			return nil, errors.New("jwt密钥未初始化")
		}
		return jwtSecret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // 锁定签名算法，防止算法混淆攻击
		jwt.WithIssuer(JWTIssuer),                                    // 校验签发者
		jwt.WithExpirationRequired(),                                 // exp必须存在且未过期
	)
	if err != nil {
		// 映射为业务错误，保持上层错误语义不变
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, common.ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, common.ErrTokenSignature
		case errors.Is(err, jwt.ErrTokenInvalidIssuer):
			return nil, common.ErrTokenIssuer
		default:
			return nil, common.ErrTokenInvalid
		}
	}
	return claims, nil
}
