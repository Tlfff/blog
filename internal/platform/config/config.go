package config

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

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
	// 1. 加载本地环境变量文件，已由运行环境注入的变量保持更高优先级
	if err := loadEnvFile(".env"); err != nil {
		return nil, err
	}

	// 2. 读取配置文件并展开环境变量占位符
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	expandedBytes, err := expandEnvironment(fileBytes)
	if err != nil {
		return nil, err
	}

	// 3. 将展开后的 YAML 反序列化为运行配置
	var cfg Config
	if err := yaml.Unmarshal(expandedBytes, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return &cfg, nil
}

// loadEnvFile 加载指定的环境变量文件，文件不存在时允许继续启动。
func loadEnvFile(filePath string) error {
	// 1. 打开环境变量文件；容器和 Kubernetes 通常直接注入变量，因此允许文件不存在
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("打开环境变量文件失败: %w", err)
	}
	defer file.Close()

	// 2. 逐行解析 KEY=VALUE，仅在运行环境尚未设置该变量时写入
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("环境变量文件第 %d 行缺少等号", lineNumber)
		}
		key = strings.TrimSpace(key)
		if !isValidEnvironmentName(key) {
			return fmt.Errorf("环境变量文件第 %d 行变量名非法", lineNumber)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		parsedValue, err := parseEnvironmentValue(strings.TrimSpace(value))
		if err != nil {
			return fmt.Errorf("解析环境变量文件第 %d 行失败: %w", lineNumber, err)
		}
		if err := os.Setenv(key, parsedValue); err != nil {
			return fmt.Errorf("设置环境变量 %s 失败: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取环境变量文件失败: %w", err)
	}
	return nil
}

// expandEnvironment 展开配置内容中的环境变量，并拒绝缺失的变量。
func expandEnvironment(content []byte) ([]byte, error) {
	// 1. 展开变量时记录未由系统环境或 .env 提供的变量名
	missingVariables := make(map[string]struct{})
	expanded := os.Expand(string(content), func(name string) string {
		value, exists := os.LookupEnv(name)
		if !exists {
			missingVariables[name] = struct{}{}
			return ""
		}
		return value
	})
	if len(missingVariables) == 0 {
		return []byte(expanded), nil
	}

	// 2. 稳定输出缺失变量列表，便于快速定位部署配置问题
	names := make([]string, 0, len(missingVariables))
	for name := range missingVariables {
		names = append(names, name)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("缺少必需环境变量: %s", strings.Join(names, ", "))
}

// parseEnvironmentValue 解析 .env 中的普通值和双引号值。
func parseEnvironmentValue(value string) (string, error) {
	// 1. 保留未加引号的值，并去除单引号包裹
	if len(value) < 2 {
		return value, nil
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1], nil
	}

	// 2. 使用 Go 字符串规则解析双引号转义内容
	if value[0] == '"' && value[len(value)-1] == '"' {
		parsedValue, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("双引号值格式非法: %w", err)
		}
		return parsedValue, nil
	}
	return value, nil
}

// isValidEnvironmentName 判断名称是否符合常用环境变量命名规则。
func isValidEnvironmentName(name string) bool {
	// 1. 首字符必须是字母或下划线
	if name == "" || !isEnvironmentNameStart(name[0]) {
		return false
	}

	// 2. 后续字符仅允许字母、数字和下划线
	for i := 1; i < len(name); i++ {
		char := name[i]
		if !isEnvironmentNameStart(char) && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

// isEnvironmentNameStart 判断字符能否作为环境变量名称的字母部分。
func isEnvironmentNameStart(char byte) bool {
	// 1. 接受大小写英文字母和下划线
	return char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}
