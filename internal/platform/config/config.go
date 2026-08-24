package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

// Config 表示博客系统完整运行配置。
type Config struct {
	Database   Database  `yaml:"database"`   // MySQL 配置
	Redis      Redis     `yaml:"redis"`      // Redis 配置
	Mongodb    Mongodb   `yaml:"mongodb"`    // MongoDB 配置
	Kafka      Kafka     `yaml:"kafka"`      // Kafka 配置
	JWT        JWT       `yaml:"jwt"`        // 用户 JWT 配置
	OpenJWT    OpenJWT   `yaml:"openjwt"`    // 开放 gRPC JWT 配置
	GRPC       GRPC      `yaml:"grpc"`       // gRPC Server 配置
	ThirdParty []Partner `yaml:"thirdparty"` // 三方合作方配置
	OSS        OSS       `yaml:"oss"`        // MinIO 对象存储配置
}

// JWT 表示用户 JWT 配置。
type JWT struct {
	Secret string `yaml:"secret"` // JWT 签名密钥
}

// OpenJWT 二方服务专用JWT配置，与C端用户JWT完全隔离
type OpenJWT struct {
	Secret string `yaml:"secret"` // 开放 gRPC JWT 签名密钥
}

// GRPC 表示 gRPC Server 配置。
type GRPC struct {
	Port string `yaml:"port"` // gRPC Server 监听端口
}

// Partner 三方合作方密钥配置，access_key_id 为凭证身份，secret_key 为共享密钥
type Partner struct {
	PartnerID   string `yaml:"partner_id"`    // 组织标识，仅用于统计
	AccessKeyID string `yaml:"access_key_id"` // 凭证身份
	SecretKey   string `yaml:"secret_key"`    // 共享密钥
}

// Database 表示 MySQL 连接配置。
type Database struct {
	Username string `yaml:"username"` // MySQL 用户名
	Password string `yaml:"password"` // MySQL 密码
	Host     string `yaml:"host"`     // MySQL 主机地址
	Port     int    `yaml:"port"`     // MySQL 端口
	DBName   string `yaml:"dbname"`   // MySQL 数据库名称
}

// Redis 表示 Redis 连接配置。
type Redis struct {
	Addr     string `yaml:"addr"`     // Redis 地址
	DB       int    `yaml:"db"`       // Redis 数据库编号
	Password string `yaml:"password"` // Redis 密码
}

// Mongodb 表示 MongoDB 连接配置。
type Mongodb struct {
	Host     string `yaml:"host"`     // MongoDB 主机地址
	Port     int    `yaml:"port"`     // MongoDB 端口
	DBName   string `yaml:"dbname"`   // MongoDB 数据库名称
	Username string `yaml:"username"` // MongoDB 用户名
	Password string `yaml:"password"` // MongoDB 密码
}

// OSS 表示 MinIO 对象存储配置。
type OSS struct {
	Endpoint        string   `yaml:"endpoint"`          // MinIO 服务地址
	AccessKeyID     string   `yaml:"access_key_id"`     // MinIO 访问密钥 ID
	SecretAccessKey string   `yaml:"secret_access_key"` // MinIO 访问密钥
	Bucket          string   `yaml:"bucket"`            // 对象存储桶名称
	UseSSL          bool     `yaml:"use_ssl"`           // 是否使用 HTTPS
	PublicDomain    string   `yaml:"public_domain"`     // 对象公开访问域名
	AllowedExts     []string `yaml:"allowed_exts"`      // 允许上传的文件扩展名
}

// LoadConfig 从指定 YAML 文件加载运行配置。
func LoadConfig(filePath string) (*Config, error) {
	// 读取磁盘上的 yaml 文件
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	// 利用 yaml 库，把文件字节流“反序列化”到 cfg 结构体中
	// 此时，`yaml:"username"` 标签开始起作用，精准匹配数据
	err = yaml.Unmarshal(fileBytes, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
