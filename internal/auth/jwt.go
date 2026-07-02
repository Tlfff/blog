package auth

import (
	"blog/internal/common"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	JWTSecretKey  = "7e29acf58d421036b897f31ce05da74918bf2c45e6839021ad654872f90ac165" //JWT的签名密钥，应该用一个足够复杂的字符串，并且放在配置文件中
	JWTIssuer     = "blog"                                                             //JWT的签发者
	JWTExpireTime = time.Hour * 24                                                     //JWT的过期时间，这里设置为24小时
)

// 预编码固定JWT Header，避免每次重复序列化
var encodedHeader string

func init() {
	headerMap := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}
	raw, _ := json.Marshal(headerMap)
	encodedHeader = base64UrlEncode(raw)
}

// JWT专用无填充Base64编码
func base64UrlEncode(b []byte) string {
	// Go的标准库提供的Base64编码会使用"+"和"/"字符，并且在末尾添加"="作为填充。JWT要求使用URL安全的Base64编码，即将"+"替换为"-"，将"/"替换为"_"，并且去掉末尾的"="。
	s := base64.StdEncoding.EncodeToString(b)
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	return strings.TrimRight(s, "=")
}

// JWT专用无填充Base64解码
func base64UrlDecode(s string) ([]byte, error) {
	// JWT的Base64解码需要先将URL安全的字符替换回原来的字符，并且根据长度添加适当的填充。
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	// 把原本删去的末尾=加上
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}

// 生成Token
func GenerateToken(phone string, role int8, userID uint64) (string, error) {
	// 1. 组装数据
	claims := &Claims{
		UserID: userID,
		Phone:  phone,
		Role:   role,
		Iss:    JWTIssuer,
		Iat:    time.Now().Unix(),
		Exp:    time.Now().Add(JWTExpireTime).Unix(),
	}
	// 2. 序列化并编码 Payload
	payloadRaw, err := json.Marshal(claims) //将Claims结构体序列化为JSON格式的字节切片
	if err != nil {
		return "", err
	}
	encodedPayload := base64UrlEncode(payloadRaw) //对序列化后的字节切片进行Base64编码，得到JWT的Payload部分
	// 3. 生成签名
	signSource := encodedHeader + "." + encodedPayload //将JWT的Header和Payload部分用点号连接起来，作为签名的原始数据
	h := hmac.New(sha256.New, []byte(JWTSecretKey))    //创建一个新的HMAC对象，使用SHA256算法和预定义的密钥，生成的签名是二进制字节流，直接传给前端会不可见
	h.Write([]byte(signSource))
	// 4. 编码签名
	sign := base64UrlEncode(h.Sum(nil)) //计算HMAC签名，并对结果进行Base64编码，得到JWT的Signature部分
	// 5. 三段拼接
	return fmt.Sprintf("%s.%s.%s", encodedHeader, encodedPayload, sign), nil //将Header、Payload和Signature部分用点号连接起来，形成完整的JWT字符串，并返回
}

// 解析和验证Token
func VerifyToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		log.Println("token错误,应该包含3部分")
		return nil, errors.New(common.ErrTokenInvalid.Error())
	}
	encodedHeader, encodedPayload, encodedSign := parts[0], parts[1], parts[2]
	// 验证签名
	// 1、重新计算签名
	signSource := encodedHeader + "." + encodedPayload
	h := hmac.New(sha256.New, []byte(JWTSecretKey))
	// 2、计算预期的签名
	h.Write([]byte(signSource))
	expectedSign := base64UrlEncode(h.Sum(nil))
	if hmac.Equal([]byte(encodedSign), []byte(expectedSign)) == false {
		log.Println("token签名无效")
		return nil, errors.New(common.ErrTokenSignature.Error())
	}
	// 解析Payload
	payloadRaw, err := base64UrlDecode(encodedPayload)
	if err != nil {
		log.Printf("patload 解析失败: %s", err)
		return nil, errors.New(common.ErrTokenInvalid.Error())
	}
	// 将解码后的字节切片反序列化为Claims结构体
	var claims Claims
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		log.Printf("payload 反序列化失败: %s", err)
		return nil, errors.New(common.ErrTokenInvalid.Error())
	}
	// 验证签发者
	if claims.Iss != JWTIssuer {
		log.Println("token的签发者无效")
		return nil, errors.New(common.ErrTokenIssuer.Error())
	}
	// 验证过期时间
	if time.Now().Unix() > claims.Exp {
		log.Println("token已过期")
		return nil, errors.New(common.ErrTokenExpired.Error())
	}
	return &claims, nil
}
