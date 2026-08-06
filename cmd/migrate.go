package cmd

import (
	"blog/config"
	"blog/pkg/database"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "通过 scripts/mysql 目录下的 SQL 文件初始化数据库",
	Long:  `自动识别并读取 scripts/mysql 目录下的所有 .sql 文件，优先建立数据库，再按顺序初始化所有业务表结构`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(" 开始从本地脚本目录执行数据库迁移...")

		// 1. 加载配置文件
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			fmt.Printf("[error] 加载配置文件失败：%v\n", err)
			return
		}

		// 2. 依然是先不指定数据库建立基础连接，防止因为库不存在而报错
		db, err := database.NewMySQLClient(cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, "")
		if err != nil {
			fmt.Printf("[error] 建立 MySQL 基础连接失败：%v\n", err)
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			fmt.Printf("[error] 获取底层 sql.DB 实例失败：%v\n", err)
			return
		}
		defer sqlDB.Close()

		sqlDir := "scripts/mysql"

		// 3. 第一步：专门去读并执行 database_blog.sql（建库）
		initDBFile := filepath.Join(sqlDir, "database_blog.sql")
		fmt.Printf("正在读取建库脚本: %s ...\n", initDBFile)
		dbSqlBytes, err := os.ReadFile(initDBFile)
		if err != nil {
			fmt.Printf("[error] 读取建库脚本失败，请确认文件是否存在: %v\n", err)
			return
		}

		// 执行建库（有些脚本里自带了 USE 语句，我们直接执行）
		if err := executeMultiSQL(sqlDB, string(dbSqlBytes)); err != nil {
			fmt.Printf("[error] 执行建库脚本失败: %v\n", err)
			return
		}
		fmt.Println("数据库环境检测/初始化成功！")

		// 4. 第二步：显式切换到你在 config.yaml 里指定的数据库名
		_, err = sqlDB.Exec(fmt.Sprintf("USE %s;", cfg.Database.DBName))
		if err != nil {
			fmt.Printf("[error] 切换至数据库 [%s] 失败: %v\n", cfg.Database.DBName, err)
			return
		}

		// 5. 第三步：扫描目录下其他的建表 sql 并批量执行
		files, err := os.ReadDir(sqlDir)
		if err != nil {
			fmt.Printf("[error] 读取 SQL 目录失败: %v\n", err)
			return
		}

		for _, file := range files {
			// 跳过目录本身和已经执行过的建库文件
			if file.IsDir() || file.Name() == "database_blog.sql" || !strings.HasSuffix(file.Name(), ".sql") {
				continue
			}

			filePath := filepath.Join(sqlDir, file.Name())
			fmt.Printf("正在执行数据表脚本: %s ...\n", file.Name())

			tableSqlBytes, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("[error] 读取文件 %s 失败: %v\n", file.Name(), err)
				continue
			}

			if err := executeMultiSQL(sqlDB, string(tableSqlBytes)); err != nil {
				fmt.Printf("[error] 脚本 %s 执行失败: %v\n", file.Name(), err)
				return
			}
		}

		fmt.Println("恭喜！scripts/mysql 目录下的所有 SQL 脚本已全部顺利执行完毕！")
	},
}

// 辅助函数：处理单个 .sql 文件中可能包含的多条 SQL 语句（以分号分割）
func executeMultiSQL(db *sql.DB, sqlContent string) error {
	// 将多条语句通过分号拆开（同时清理掉前后空格和换行）
	queries := strings.Split(sqlContent, ";")
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		// 执行单条语句
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
