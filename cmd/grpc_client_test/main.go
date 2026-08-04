// 临时测试客户端：验证 gRPC 二方(JWT)/三方(HMAC) 接口与日志输出
// 用法: go run ./cmd/grpc_client_test
package main

import (
	"blog/config"
	"blog/internal/auth"
	"context"
	"fmt"
	"strconv"
	"time"

	userv1 "blog/gen/user"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const addr = "127.0.0.1:9100"

func main() {
	// 初始化二方JWT密钥（与服务端同一份配置）
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		panic(err)
	}
	auth.InitOpenJWT(cfg.OpenJWT.Secret)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := userv1.NewUserServiceClient(conn)

	// 1. 二方：JWT 鉴权
	if err := testInternal(client); err != nil {
		fmt.Println("[二方] 失败:", err)
	} else {
		fmt.Println("[二方] 成功")
	}

	// 2. 三方：HMAC 签名鉴权
	if err := testExternal(client); err != nil {
		fmt.Println("[三方] 失败:", err)
	} else {
		fmt.Println("[三方] 成功")
	}

	// 3. 无凭证：应被拒绝（验证日志能记录失败）
	_, err = client.GetPublicUserInfo(context.Background(), &userv1.GetUserInfoRequest{UserId: 1})
	fmt.Println("[无凭证] 应失败:", err != nil)
}

// testInternal 二方调用：JWT + GetUserBasicInfo
func testInternal(client userv1.UserServiceClient) error {
	// 生成二方 JWT（service_id + team_id），注意需与服务端同一 secret
	token, err := auth.OpenGenerateToken("search-service", "team-a")
	if err != nil {
		return err
	}
	ctx := metadata.AppendToOutgoingContext(context.Background(),
		"authorization", "Bearer "+token,
		"x-trace-id", "trace-"+uuid.NewString()[:8],
	)
	resp, err := client.GetUserBasicInfo(ctx, &userv1.GetUserBasicInfoRequest{UserId: 1})
	if err != nil {
		return err
	}
	fmt.Printf("[二方响应] user_id=%d nickname=%s avatar=%s\n", resp.UserId, resp.Nickname, resp.Avatar)
	return nil
}

// testExternal 三方调用：HMAC 签名 + GetPublicUserInfo
func testExternal(client userv1.UserServiceClient) error {
	accessKeyID := "ak-partner-a"
	secretKey := "sk-partner-a-secret-key"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := uuid.NewString()

	req := &userv1.GetUserInfoRequest{UserId: 1}
	bodyBytes, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	bodyHash := auth.BuildBodyHash(bodyBytes)

	method := "/blogopen.v1.UserService/GetPublicUserInfo"
	signature := auth.HMACSign(secretKey, accessKeyID, method, timestamp, nonce, bodyHash)

	ctx := metadata.AppendToOutgoingContext(context.Background(),
		"x-access-key-id", accessKeyID,
		"x-signature", signature,
		"x-timestamp", timestamp,
		"x-nonce", nonce,
		"x-trace-id", "trace-"+uuid.NewString()[:8],
	)
	resp, err := client.GetPublicUserInfo(ctx, req)
	if err != nil {
		return err
	}
	fmt.Printf("[三方响应] id=%d nickname=%s avatar=%s\n", resp.Id, resp.Nickname, resp.Avatar)
	return nil
}
