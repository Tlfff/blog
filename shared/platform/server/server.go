// Package server 提供内部 gRPC 服务的启动、健康检查与优雅退出。
package server

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	platformgrpc "blog/shared/platform/grpc"
)

// Run 启动 gRPC 服务并阻塞直到收到退出信号或服务异常退出。
func Run(addr, serviceName string, allowedIDs map[string]bool, register func(*grpc.Server)) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s := grpc.NewServer(grpc.ChainUnaryInterceptor(platformgrpc.ServerAuthInterceptor(allowedIDs)))
	register(s)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("%s gRPC 服务启动，监听 %s", serviceName, addr)
		serveErr <- s.Serve(lis)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		return err
	case <-sig:
		log.Printf("收到退出信号，正在优雅停止 %s", serviceName)
		healthServer.Shutdown()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stopped := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-ctx.Done():
			s.Stop()
		}
		return nil
	}
}
