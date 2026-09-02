// Package bootstrap 负责初始化和关闭进程级技术资源。
package bootstrap

import (
	platformconfig "blog/internal/platform/config"
	platformdatabase "blog/internal/platform/database"
	platformelasticsearch "blog/internal/platform/elasticsearch"
	platformkafka "blog/internal/platform/kafka"
	platformoss "blog/internal/platform/oss"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// ResourceOptions 描述当前进程需要初始化的技术资源。
type ResourceOptions struct {
	MySQL                 bool // 是否初始化 MySQL
	MongoDB               bool // 是否初始化 MongoDB
	Redis                 bool // 是否初始化 Redis
	Kafka                 bool // 是否初始化 Kafka
	OSS                   bool // 是否初始化 MinIO
	Elasticsearch         bool // 是否初始化 Elasticsearch
	AllowMongoDBInitError bool // MongoDB 初始化失败时是否记录日志后继续
}

// Resources 保存当前进程共享的技术客户端。
type Resources struct {
	MySQL         *gorm.DB                      // MySQL GORM 客户端
	MongoDB       *mongo.Database               // MongoDB 数据库客户端
	Redis         *redis.Client                 // Redis 客户端
	Kafka         *platformkafka.Client         // Kafka 客户端
	OSS           *platformoss.MinioClient      // MinIO 客户端
	Elasticsearch *platformelasticsearch.Client // Elasticsearch 客户端
	closeOnce     sync.Once                     // 保证关闭逻辑只执行一次
	closeErr      error                         // 关闭过程中记录的错误
}

// NewResources 按固定顺序初始化当前进程所需的技术资源。
//
// 参数说明：
//   - cfg：平台运行配置，不能为空。
//   - options：当前进程需要初始化的资源集合。
//
// 初始化顺序固定为 MySQL、MongoDB、Redis、Kafka、MinIO、Elasticsearch；已初始化资源失败时会尝试回收。
func NewResources(cfg *platformconfig.Config, options ResourceOptions) (*Resources, error) {
	if cfg == nil {
		return nil, errors.New("平台配置不能为空")
	}

	resources := &Resources{}
	cleanupOnError := func(err error) (*Resources, error) {
		_ = resources.Close()
		return nil, err
	}

	// 1. 初始化 MySQL
	if options.MySQL {
		db, err := platformdatabase.NewMySQLClient(
			cfg.Database.Username,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName,
		)
		if err != nil {
			return cleanupOnError(fmt.Errorf("初始化 MySQL 失败: %w", err))
		}
		if db == nil {
			return cleanupOnError(errors.New("初始化 MySQL 失败: 返回空数据库连接"))
		}
		resources.MySQL = db
	}

	// 2. 初始化 MongoDB
	if options.MongoDB {
		mongodb, err := platformdatabase.NewMongoDBClient(
			cfg.Mongodb.Username,
			cfg.Mongodb.Password,
			cfg.Mongodb.Host,
			cfg.Mongodb.DBName,
			cfg.Mongodb.Port,
		)
		if err != nil {
			if !options.AllowMongoDBInitError {
				return cleanupOnError(fmt.Errorf("初始化 MongoDB 失败: %w", err))
			}
			log.Printf("[WARN] 初始化 MongoDB 失败，按当前入口策略继续启动: %v", err)
		} else {
			resources.MongoDB = mongodb
		}
	}

	// 3. 初始化 Redis
	if options.Redis {
		rdb, err := platformdatabase.NewRedisClient(cfg.Redis)
		if err != nil {
			return cleanupOnError(fmt.Errorf("初始化 Redis 失败: %w", err))
		}
		resources.Redis = rdb
	}

	// 4. 初始化 Kafka
	if options.Kafka {
		client, err := platformkafka.NewClient(cfg.Kafka)
		if err != nil {
			return cleanupOnError(fmt.Errorf("初始化 Kafka 失败: %w", err))
		}
		resources.Kafka = client
	}

	// 5. 初始化 MinIO
	if options.OSS {
		client, err := platformoss.NewMinioClient(
			cfg.OSS.Endpoint,
			cfg.OSS.AccessKeyID,
			cfg.OSS.SecretAccessKey,
			cfg.OSS.Bucket,
			cfg.OSS.UseSSL,
		)
		if err != nil {
			return cleanupOnError(fmt.Errorf("初始化 MinIO 失败: %w", err))
		}
		resources.OSS = client
	}

	// 6. 初始化 Elasticsearch
	if options.Elasticsearch {
		client, err := platformelasticsearch.NewClient(cfg.Elasticsearch)
		if err != nil {
			return cleanupOnError(fmt.Errorf("初始化 Elasticsearch 失败: %w", err))
		}
		resources.Elasticsearch = client
	}

	return resources, nil
}

// Close 按资源生命周期关闭当前进程持有的客户端。
func (r *Resources) Close() error {
	if r == nil {
		return nil
	}

	r.closeOnce.Do(func() {
		var errs []error

		// 1. 关闭 Elasticsearch
		if r.Elasticsearch != nil {
			if err := r.Elasticsearch.Close(); err != nil {
				errs = append(errs, fmt.Errorf("关闭 Elasticsearch 失败: %w", err))
			}
		}

		// 2. 关闭 Kafka
		if r.Kafka != nil {
			if err := r.Kafka.Close(); err != nil {
				errs = append(errs, fmt.Errorf("关闭 Kafka 失败: %w", err))
			}
		}

		// 3. 关闭 Redis
		if r.Redis != nil {
			if err := r.Redis.Close(); err != nil {
				errs = append(errs, fmt.Errorf("关闭 Redis 失败: %w", err))
			}
		}

		// 4. 关闭 MongoDB 客户端
		if r.MongoDB != nil {
			if err := r.MongoDB.Client().Disconnect(context.Background()); err != nil {
				errs = append(errs, fmt.Errorf("关闭 MongoDB 失败: %w", err))
			}
		}

		// 5. 关闭 MySQL 底层连接
		if r.MySQL != nil {
			if sqlDB, err := r.MySQL.DB(); err != nil {
				errs = append(errs, fmt.Errorf("获取 MySQL 底层连接失败: %w", err))
			} else if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("关闭 MySQL 失败: %w", err))
			}
		}

		r.closeErr = errors.Join(errs...)
	})

	return r.closeErr
}
