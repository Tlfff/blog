package interceptor

import (
	"blog/internal/infrastructure/config"
	"blog/internal/auth"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// HMAC metadata 中三方签名相关的 key
const (
	mdKeyAccessKeyID = "x-access-key-id" // 凭证身份
	mdKeySignature   = "x-signature"     // HMAC-SHA256 签名
	mdKeyTimestamp   = "x-timestamp"     // 请求时间戳
	mdKeyNonce       = "x-nonce"         // 随机数，防重放

	//  允许的时间戳偏差窗口（秒）：拒绝窗口外的请求，防止重放
	timestampWindow = 60 * time.Second
	// nonceKeyPrefix nonce 去重 key 前缀
	nonceKeyPrefix = "hmac:nonce:"
)

// 认证拦截器（三方）：校验外部合作方的 HMAC 签名
type HmacAuthInterceptor struct {
	// accessKeyID -> 合作方密钥配置（来自配置文件，不建表）
	partners map[string]config.Partner
	rdb      *redis.Client // 用于 nonce 去重
}

// 用配置文件中的合作方列表构建拦截器
func NewHmacAuthInterceptor(rdb *redis.Client, partners []config.Partner) *HmacAuthInterceptor {
	m := make(map[string]config.Partner, len(partners))
	for _, p := range partners {
		m[p.AccessKeyID] = p
	}
	return &HmacAuthInterceptor{partners: m, rdb: rdb}
}

// Unary 返回 gRPC 一元拦截器函数
func (h *HmacAuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return h.Intercept
}

// 校验三方 HMAC 签名，通过后把合作方身份注入 context
func (h *HmacAuthInterceptor) Intercept(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 1. 从 metadata 取出签名要素
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "缺少认证信息")
	}
	accessKeyID := firstMetaValue(md, mdKeyAccessKeyID)
	signature := firstMetaValue(md, mdKeySignature)
	timestamp := firstMetaValue(md, mdKeyTimestamp)
	nonce := firstMetaValue(md, mdKeyNonce)
	if accessKeyID == "" || signature == "" || timestamp == "" || nonce == "" {
		return nil, status.Error(codes.Unauthenticated, "缺少签名参数")
	}

	// 2. 时间戳窗口校验：拒绝过期或来自未来的请求，防止延迟重放
	if err := checkTimestamp(timestamp); err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	// 3. nonce 去重：同一 nonce 在窗口内只能使用一次
	if err := h.checkNonce(ctx, accessKeyID, nonce); err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	// 4. 根据 access_key_id 查出对应的 secret_key
	partner, exists := h.partners[accessKeyID]
	if !exists {
		return nil, status.Error(codes.Unauthenticated, "未知的access_key_id")
	}

	// 5. 计算请求体哈希，参与签名校验（防止请求参数被篡改）
	bodyHash, err := computeBodyHash(req)
	if err != nil {
		return nil, status.Error(codes.Internal, "无法序列化请求体")
	}

	// 6. 用相同规则重算签名并恒定时间比对
	// 签名原文：access_key_id + method_name + timestamp + nonce + request_body_hash
	if !auth.HMACVerify(partner.SecretKey, accessKeyID, info.FullMethod, timestamp, nonce, bodyHash, signature) {
		return nil, status.Error(codes.Unauthenticated, "签名校验失败")
	}

	// 7. 校验通过，注入合作方身份（ID 为凭证身份 access_key_id，Group 为组织标识）
	ctx = withIdentity(ctx, &Identity{
		Kind:  KindExternal,
		ID:    accessKeyID,
		Group: partner.PartnerID,
	})
	return handler(ctx, req)
}

// 校验时间戳是否在允许窗口内（防止延迟重放）
func checkTimestamp(timestamp string) error {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("时间戳格式错误")
	}
	now := time.Now()
	reqTime := time.Unix(ts, 0)
	if diff := now.Sub(reqTime); diff < -timestampWindow || diff > timestampWindow {
		return errors.New("请求时间戳已过期")
	}
	return nil
}

// 用 Redis SETNX 实现 nonce 去重：同一 nonce 窗口内只能成功一次
func (h *HmacAuthInterceptor) checkNonce(ctx context.Context, accessKeyID, nonce string) error {
	key := nonceKeyPrefix + accessKeyID + ":" + nonce
	ok, err := h.rdb.SetNX(ctx, key, "1", timestampWindow).Result()
	if err != nil {
		return errors.New("nonce校验服务异常")
	}
	if !ok {
		return errors.New("nonce重复使用")
	}
	return nil
}

// computeBodyHash 对 gRPC 请求体做 proto.Marshal 后计算 SHA256
// 调用方必须使用相同的序列化规则（proto3 序列化是确定性的）
func computeBodyHash(req any) (string, error) {
	msg, ok := req.(proto.Message)
	if !ok {
		return "", errors.New("请求体不是protobuf消息")
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}
	return auth.BuildBodyHash(data), nil
}

// 取 metadata 中某个 key 的第一个值（grpc metadata key 统一小写）
func firstMetaValue(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
