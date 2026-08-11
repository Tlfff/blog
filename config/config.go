package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Database   Database  `yaml:"database"`
	Redis      Redis     `yaml:"redis"`
	Mongodb    Mongodb   `yaml:"mongodb"`
	Kafka      Kafka     `yaml:"kafka"`
	JWT        JWT       `yaml:"jwt"`
	OpenJWT    OpenJWT   `yaml:"openjwt"`
	GRPC       GRPC      `yaml:"grpc"`
	ThirdParty []Partner `yaml:"thirdparty"`
	OSS        OSS       `yaml:"oss"`
}

type JWT struct {
	Secret string `yaml:"secret"` // JWT签名密钥
}

// OpenJWT 二方服务专用JWT配置，与C端用户JWT完全隔离
type OpenJWT struct {
	Secret string `yaml:"secret"`
}

type GRPC struct {
	Port string `yaml:"port"`
}

// Partner 三方合作方密钥配置，access_key_id 为凭证身份，secret_key 为共享密钥
type Partner struct {
	PartnerID   string `yaml:"partner_id"`    // 组织标识，仅用于统计
	AccessKeyID string `yaml:"access_key_id"` // 凭证身份
	SecretKey   string `yaml:"secret_key"`    // 共享密钥
}
type Database struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"dbname"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	DB       int    `yaml:"db"`
	Password string `yaml:"password"`
}
type Mongodb struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"dbname"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type OSS struct {
	Endpoint        string   `yaml:"endpoint"`
	AccessKeyID     string   `yaml:"access_key_id"`
	SecretAccessKey string   `yaml:"secret_access_key"`
	Bucket          string   `yaml:"bucket"`
	UseSSL          bool     `yaml:"use_ssl"`
	PublicDomain    string   `yaml:"public_domain"`
	AllowedExts     []string `yaml:"allowed_exts"` // 允许上传的文件扩展名
}

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
