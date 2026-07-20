package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Database Database `yaml:"database"`
	Redis    Redis    `yaml:"redis"`
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
