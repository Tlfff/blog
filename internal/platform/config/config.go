// Package config 负责加载并解析应用的 YAML 配置。
package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

// Config 是应用的顶层配置聚合，对应配置文件的根节点。
type Config struct {
	Database   Database  `yaml:"database"`   // MySQL 数据库配置
	Redis      Redis     `yaml:"redis"`      // Redis 缓存配置
	Mongodb    Mongodb   `yaml:"mongodb"`    // MongoDB 配置，用于存储通知
	Kafka      Kafka     `yaml:"kafka"`      // Kafka 消息队列配置
	JWT        JWT       `yaml:"jwt"`        // C端用户JWT配置
	OpenJWT    OpenJWT   `yaml:"openjwt"`    // 二方服务JWT配置
	GRPC       GRPC      `yaml:"grpc"`       // 开放 gRPC 服务配置
	ThirdParty []Partner `yaml:"thirdparty"` // 三方合作方密钥列表
	OSS        OSS       `yaml:"oss"`        // 对象存储配置
}

// JWT 是 C 端用户登录令牌的签名配置。
type JWT struct {
	Secret string `yaml:"secret"` // JWT签名密钥
}

// OpenJWT 二方服务专用JWT配置，与C端用户JWT完全隔离
type OpenJWT struct {
	Secret string `yaml:"secret"` // 二方服务JWT签名密钥
}

// GRPC 是开放 gRPC 服务的监听配置。
type GRPC struct {
	Port string `yaml:"port"` // gRPC 服务监听端口
}

// Partner 三方合作方密钥配置，access_key_id 为凭证身份，secret_key 为共享密钥
type Partner struct {
	PartnerID   string `yaml:"partner_id"`    // 组织标识，仅用于统计
	AccessKeyID string `yaml:"access_key_id"` // 凭证身份
	SecretKey   string `yaml:"secret_key"`    // 共享密钥
}

// Database 是 MySQL 连接配置。
type Database struct {
	Username string `yaml:"username"` // 数据库用户名
	Password string `yaml:"password"` // 数据库密码
	Host     string `yaml:"host"`     // 数据库主机地址
	Port     int    `yaml:"port"`     // 数据库端口
	DBName   string `yaml:"dbname"`   // 数据库名
}

// Redis 是 Redis 连接配置。
type Redis struct {
	Addr     string `yaml:"addr"`     // Redis 地址，格式为 host:port
	DB       int    `yaml:"db"`       // Redis 逻辑库编号
	Password string `yaml:"password"` // Redis 密码，无密码时留空
}

// Mongodb 是 MongoDB 连接配置。
type Mongodb struct {
	Host     string `yaml:"host"`     // MongoDB 主机地址
	Port     int    `yaml:"port"`     // MongoDB 端口
	DBName   string `yaml:"dbname"`   // MongoDB 数据库名
	Username string `yaml:"username"` // MongoDB 用户名
	Password string `yaml:"password"` // MongoDB 密码
}

// OSS 是对象存储（MinIO）配置。
type OSS struct {
	Endpoint        string   `yaml:"endpoint"`          // 对象存储服务地址
	AccessKeyID     string   `yaml:"access_key_id"`     // 访问密钥ID
	SecretAccessKey string   `yaml:"secret_access_key"` // 访问密钥Secret
	Bucket          string   `yaml:"bucket"`            // 存储桶名称
	UseSSL          bool     `yaml:"use_ssl"`           // 是否启用HTTPS
	PublicDomain    string   `yaml:"public_domain"`     // 对外访问的公开域名，用于拼接资源URL
	AllowedExts     []string `yaml:"allowed_exts"`      // 允许上传的文件扩展名
}

// 从指定路径加载 YAML 配置并反序列化为 Config
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
