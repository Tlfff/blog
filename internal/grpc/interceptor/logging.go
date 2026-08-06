package interceptor

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	mdKeyTraceID = "x-trace-id"
	// maxLogBodyLen 请求/响应体打印的最大长度，防止日志刷屏
	maxLogBodyLen = 500
)

// LoggingInterceptor 日志拦截器（放在拦截器链最外层）
// 进入时打印方法名与请求参数；退出时打印耗时、身份与错误/响应
func LoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 1. 获取或生成链路 TraceID
	traceID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(mdKeyTraceID); len(vals) > 0 {
			traceID = vals[0]
		}
		// 把 traceID 追加进原始 metadata（保留原有字段），供下游拦截器使用
		md = metadata.Join(md, metadata.Pairs(mdKeyTraceID, traceID))
		ctx = metadata.NewIncomingContext(ctx, md)
	}
	if traceID == "" {
		traceID = uuid.New().String()
	}

	startTime := time.Now()

	// 2. 创建共享身份容器：认证拦截器（内层）会把身份写入holder，
	// 日志拦截器（外层）退出时才能读取到
	ctx = context.WithValue(ctx, identityHolderKey{}, &identityHolder{})

	// 3. 进入日志：方法名 + 请求参数
	log.Printf("[Trace-Start] ID: %s | Method: %s | Req: %s",
		traceID, info.FullMethod, truncate(protojsonString(req)))

	// 4. 执行后续拦截器链与业务 Handler
	resp, err := handler(ctx, req)

	// 5. 退出日志：耗时 + 身份 + 响应/错误
	latency := time.Since(startTime)
	identityStr := "anonymous"
	if h, ok := ctx.Value(identityHolderKey{}).(*identityHolder); ok && h.identity != nil {
		identityStr = string(h.identity.Kind) + ":" + h.identity.ID
	}
	if err != nil {
		st, _ := status.FromError(err)
		log.Printf("[Trace-End] ID: %s | Method: %s | Latency: %v | Identity: %s | Error(code=%s): %s",
			traceID, info.FullMethod, latency, identityStr, st.Code(), st.Message())
	} else {
		log.Printf("[Trace-End] ID: %s | Method: %s | Latency: %v | Identity: %s | Resp: %s",
			traceID, info.FullMethod, latency, identityStr, truncate(protojsonString(resp)))
	}
	return resp, err
}

// protojsonString 将 protobuf 消息序列化为 JSON 字符串
func protojsonString(msg any) string {
	pb, ok := msg.(proto.Message)
	if !ok {
		return ""
	}
	data, err := protojson.Marshal(pb)
	if err != nil {
		return ""
	}
	return string(data)
}

// truncate 超长内容截断，防止日志刷屏
func truncate(s string) string {
	if len(s) > maxLogBodyLen {
		return s[:maxLogBodyLen] + "...(truncated)"
	}
	return s
}
