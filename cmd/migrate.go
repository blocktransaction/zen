package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/config"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	_ "gorm.io/driver/mysql"
)

const (
	defaultMigrationsDir = "migrations"
	dbDriver             = "mysql"
)

var (
	migrationsDir string
	env           string
	configPath    string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "管理数据库迁移",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		setup()
		// 确保目录存在
		if migrationsDir == "" {
			migrationsDir = defaultMigrationsDir
		}
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			return fmt.Errorf("迁移目录不存在: %s", migrationsDir)
		}
		return nil
	},
}

// 获取配置文件
func setup() {
	config.Setup(
		configPath,
	)
}

// 初始化
func init() {
	rootCmd.AddCommand(migrateCmd)

	// 通用 flags
	migrateCmd.PersistentFlags().StringVarP(&migrationsDir, "dir", "d", defaultMigrationsDir, "迁移目录 (默认: migrations)")
	migrateCmd.PersistentFlags().StringVarP(&env, "env", "e", "test", "数据库环境 (test/prod)")
	migrateCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/", "配置目录(默认：config)")

	// up
	migrateCmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "迁移到最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGooseCommand("up", args)
		},
	})

	// down
	migrateCmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "回滚一个版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGooseCommand("down", args)
		},
	})

	// status
	migrateCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "显示迁移状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGooseCommand("status", args)
		},
	})

	// create
	migrateCmd.AddCommand(&cobra.Command{
		Use:   "create [name]",
		Short: "创建一个新的 SQL 迁移文件",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fullArgs := append(args, "sql")
			return runGooseCommand("create", fullArgs)
		},
	})
}

// 获取 DSN
func getDSN(env string) string {
	switch env {
	case constant.Test:
		return config.MysqlConfig.Test.Dsn
	case constant.Prod:
		return config.MysqlConfig.Prod.Dsn
	default:
		log.Fatalf("未知环境: %s", env)
		return ""
	}
}

// 执行 goose 命令
func runGooseCommand(command string, args []string) error {
	dsn := getDSN(env)

	db, err := goose.OpenDBWithDriver(dbDriver, dsn)
	if err != nil {
		return fmt.Errorf("goose: 无法打开数据库: %v", err)
	}
	defer db.Close()

	if err := goose.RunContext(context.Background(), command, db, migrationsDir, args...); err != nil {
		return fmt.Errorf("goose run %v failed: %v", command, err)
	}

	return nil
}

// 应用启动时自动执行迁移
func runMigrations() error {
	dsn := getDSN(env)

	db, err := goose.OpenDBWithDriver(dbDriver, dsn)
	if err != nil {
		return fmt.Errorf("goose: 无法打开数据库: %v", err)
	}
	defer db.Close()

	if _, err := goose.EnsureDBVersion(db); err != nil {
		return fmt.Errorf("goose: 无法确保数据库版本: %v", err)
	}

	fmt.Printf("正在进行数据库迁移 (env=%s, dir=%s)...\n", env, migrationsDir)
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose: up 失败: %v", err)
	}
	fmt.Println("数据库迁移成功。")
	return nil
}
