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
