// Package bootstrap 收敛配置加载与基础设施客户端初始化。
package bootstrap

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/infrastructure/config"
	"blog/pkg/database"
	"blog/pkg/kafka"
	"blog/pkg/oss"
	iputil "blog/pkg/util/ip"
	"log"
	"os"
	"path/filepath"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// InitValidator 注册自定义校验规则。
func InitValidator() {
	common.InitValidator()
}

// InitIPSearcher 基于当前工作目录初始化 ip2region 离线库。
func InitIPSearcher() error {
	// 1. 取当前工作目录作为离线库的基准路径
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	// 2. 拼出 ip2region.xdb 路径并加载
	return iputil.InitIPSearcher(filepath.Join(dir, "pkg/resource/ip2region.xdb"))
}

// CloseIP 释放 ip2region 资源。
func CloseIP() {
	iputil.Close()
}

// NewMySQL 初始化 MySQL/GORM 客户端。
func NewMySQL(cfg *config.Config) (*gorm.DB, error) {
	return database.NewMySQLClient(
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)
}

// NewMongoDB 初始化 MongoDB 客户端。
func NewMongoDB(cfg *config.Config) (*mongo.Database, error) {
	return database.NewMongoDBClient(
		cfg.Mongodb.Username,
		cfg.Mongodb.Password,
		cfg.Mongodb.Host,
		cfg.Mongodb.DBName,
		cfg.Mongodb.Port,
	)
}

// NewRedis 初始化 Redis 客户端。
func NewRedis(cfg *config.Config) (*redis.Client, error) {
	return database.NewRedisClient(cfg.Redis)
}

// NewKafka 初始化 Kafka 客户端。
func NewKafka(cfg *config.Config) (*kafka.Client, error) {
	return kafka.NewClient(cfg.Kafka)
}

// NewOSS 初始化 MinIO 客户端。
func NewOSS(cfg *config.Config) (*oss.MinioClient, error) {
	return oss.NewMinioClient(
		cfg.OSS.Endpoint,
		cfg.OSS.AccessKeyID,
		cfg.OSS.SecretAccessKey,
		cfg.OSS.Bucket,
		cfg.OSS.UseSSL,
	)
}

// InitOpenJWT 注入开放 gRPC 二方 JWT 密钥。
func InitOpenJWT(secret string) {
	auth.InitOpenJWT(secret)
}

// MustInit 在初始化失败时直接终止进程，避免把 nil 客户端继续向下传递。
func MustInit(step string, err error) {
	if err != nil {
		log.Fatalf("[error] %s 初始化失败: %v", step, err)
	}
}
