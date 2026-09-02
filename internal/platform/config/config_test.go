package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadEnvFile 验证 .env 解析和运行环境优先级。
func TestLoadEnvFile(t *testing.T) {
	// 1. 准备不会与开发机环境冲突的变量和临时 .env 文件
	fileVariable := "BLOG_CONFIG_TEST_FILE_VALUE"
	existingVariable := "BLOG_CONFIG_TEST_EXISTING_VALUE"
	quotedVariable := "BLOG_CONFIG_TEST_QUOTED_VALUE"
	for _, name := range []string{fileVariable, existingVariable, quotedVariable} {
		originalValue, existed := os.LookupEnv(name)
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(name, originalValue)
				return
			}
			_ = os.Unsetenv(name)
		})
		_ = os.Unsetenv(name)
	}
	if err := os.Setenv(existingVariable, "runtime-value"); err != nil {
		t.Fatalf("设置已有环境变量失败: %v", err)
	}
	envPath := filepath.Join(t.TempDir(), ".env")
	envContent := strings.Join([]string{
		"# 测试环境变量",
		fileVariable + "=file-value",
		existingVariable + "=file-value",
		quotedVariable + "=\"quoted value\"",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
		t.Fatalf("写入临时环境变量文件失败: %v", err)
	}

	// 2. 加载文件并确认文件值生效、已有运行环境不被覆盖
	if err := loadEnvFile(envPath); err != nil {
		t.Fatalf("加载环境变量文件失败: %v", err)
	}
	if value := os.Getenv(fileVariable); value != "file-value" {
		t.Fatalf("文件环境变量不符合预期: %q", value)
	}
	if value := os.Getenv(existingVariable); value != "runtime-value" {
		t.Fatalf("已有环境变量被意外覆盖: %q", value)
	}
	if value := os.Getenv(quotedVariable); value != "quoted value" {
		t.Fatalf("双引号环境变量解析不符合预期: %q", value)
	}
}

// TestExpandEnvironment 验证配置占位符展开和缺失变量检查。
func TestExpandEnvironment(t *testing.T) {
	// 1. 设置可用变量并验证正常展开
	t.Setenv("BLOG_CONFIG_TEST_DATABASE_USER", "bloguser")
	expanded, err := expandEnvironment([]byte(`username: "${BLOG_CONFIG_TEST_DATABASE_USER}"`))
	if err != nil {
		t.Fatalf("展开环境变量失败: %v", err)
	}
	if string(expanded) != `username: "bloguser"` {
		t.Fatalf("展开结果不符合预期: %s", expanded)
	}

	// 2. 验证缺失变量会返回清晰错误
	_, err = expandEnvironment([]byte(`password: "${BLOG_CONFIG_TEST_MISSING_PASSWORD}"`))
	if err == nil || !strings.Contains(err.Error(), "BLOG_CONFIG_TEST_MISSING_PASSWORD") {
		t.Fatalf("缺失变量错误不符合预期: %v", err)
	}
}

// TestLoadConfig 验证环境变量展开后的 YAML 能正确加载为配置。
func TestLoadConfig(t *testing.T) {
	// 1. 准备最小配置和运行环境变量
	t.Setenv("BLOG_CONFIG_TEST_MYSQL_USER", "config-user")
	t.Setenv("BLOG_CONFIG_TEST_MYSQL_PASSWORD", "config-password")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configContent := `
database:
  username: "${BLOG_CONFIG_TEST_MYSQL_USER}"
  password: "${BLOG_CONFIG_TEST_MYSQL_PASSWORD}"
  host: "127.0.0.1"
  port: 3306
  dbname: "blog"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("写入临时配置文件失败: %v", err)
	}

	// 2. 加载配置并核对敏感字段来自运行环境
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Database.Username != "config-user" || cfg.Database.Password != "config-password" {
		t.Fatalf("数据库环境变量未正确加载: %+v", cfg.Database)
	}
}

// TestSearchConfigDefaults 验证搜索相关配置的默认值和边界修正。
func TestSearchConfigDefaults(t *testing.T) {
	// 1. 验证 Elasticsearch 默认配置
	var elasticsearch Elasticsearch
	if elasticsearch.GetIndexAlias() != "article_search" {
		t.Fatalf("默认索引别名不符合预期: %q", elasticsearch.GetIndexAlias())
	}
	if elasticsearch.GetRequestTimeoutMS() != 3000 {
		t.Fatalf("默认 Elasticsearch 超时时间不符合预期: %d", elasticsearch.GetRequestTimeoutMS())
	}

	// 2. 验证 Canal 默认配置和最大退避边界
	canal := Canal{ReconnectMinWaitMS: 20000, ReconnectMaxWaitMS: 1000}
	if canal.GetBatchSize() != 100 || canal.GetEmptyWaitMS() != 500 {
		t.Fatalf("默认 Canal 批次配置不符合预期: batch=%d wait=%d", canal.GetBatchSize(), canal.GetEmptyWaitMS())
	}
	if canal.GetReconnectMaxWaitMS() != defaultCanalReconnectMaxWaitMS {
		t.Fatalf("非法最大重连等待时间未回退默认值: %d", canal.GetReconnectMaxWaitMS())
	}
}

// TestSearchConfigValidate 验证搜索连接必填配置检查。
func TestSearchConfigValidate(t *testing.T) {
	// 1. 验证 Elasticsearch 缺少地址时拒绝启动
	if err := (Elasticsearch{}).Validate(); err == nil || !strings.Contains(err.Error(), "地址") {
		t.Fatalf("Elasticsearch 缺少地址时错误不符合预期: %v", err)
	}

	// 2. 验证 Canal 缺少主机、端口或 destination 时拒绝启动
	invalidConfigs := []Canal{
		{Port: 11111, Destination: "article_search"},
		{Host: "127.0.0.1", Destination: "article_search"},
		{Host: "127.0.0.1", Port: 11111},
	}
	for _, config := range invalidConfigs {
		if err := config.Validate(); err == nil {
			t.Fatalf("非法 Canal 配置未被拒绝: %+v", config)
		}
	}
	if err := (Canal{Host: "127.0.0.1", Port: 11111, Destination: "article_search"}).Validate(); err != nil {
		t.Fatalf("合法 Canal 配置被拒绝: %v", err)
	}
}

// TestLoadSearchConfig 验证搜索配置能够从 YAML 和环境变量正确加载。
func TestLoadSearchConfig(t *testing.T) {
	// 1. 准备搜索服务配置和敏感环境变量
	t.Setenv("BLOG_CONFIG_TEST_ES_USER", "search-user")
	t.Setenv("BLOG_CONFIG_TEST_ES_PASSWORD", "search-password")
	t.Setenv("BLOG_CONFIG_TEST_CANAL_PASSWORD", "canal-password")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configContent := `
elasticsearch:
  addr: "http://127.0.0.1:9200"
  username: "${BLOG_CONFIG_TEST_ES_USER}"
  password: "${BLOG_CONFIG_TEST_ES_PASSWORD}"
  index_alias: "article_search_test"
  request_timeout_ms: 2500
canal:
  host: "127.0.0.1"
  port: 11111
  username: "canal"
  password: "${BLOG_CONFIG_TEST_CANAL_PASSWORD}"
  destination: "article_search"
  batch_size: 50
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("写入搜索配置失败: %v", err)
	}

	// 2. 加载并核对连接、认证和批次配置
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载搜索配置失败: %v", err)
	}
	if cfg.Elasticsearch.Username != "search-user" || cfg.Elasticsearch.Password != "search-password" {
		t.Fatalf("Elasticsearch 认证配置未正确加载: %+v", cfg.Elasticsearch)
	}
	if cfg.Elasticsearch.GetIndexAlias() != "article_search_test" || cfg.Elasticsearch.GetRequestTimeoutMS() != 2500 {
		t.Fatalf("Elasticsearch 索引配置不符合预期: %+v", cfg.Elasticsearch)
	}
	if cfg.Canal.Password != "canal-password" || cfg.Canal.GetBatchSize() != 50 {
		t.Fatalf("Canal 配置未正确加载: %+v", cfg.Canal)
	}
}
