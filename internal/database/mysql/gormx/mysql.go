package gormx

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	engines sync.Map // 并发安全 map[string]*gorm.DB
)

// 初始化数据库：生产 + 测试
func Setup() {
	initMysql(constant.Prod)
	initMysql(constant.Test)
}

// 初始化某个环境的 MySQL
func initMysql(env string) {
	cfg := Config{
		Level:         "info",
		File:          defaultLogFile(env),
		Rotate:        true,
		MaxSize:       100,
		MaxAge:        30,
		MaxBackups:    7,
		Compress:      true,
		EnableMasking: true,
		SlowThreshold: 200 * time.Millisecond,
	}

	// Zap JSON + BatchWrite + GORM Logger
	_, gormLogger := NewLogger(cfg)

	dsn := defaultDsn(env)
	engine, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt:            true, // 使用预编译
		SkipDefaultTransaction: true, // 更高性能
		Logger:                 gormLogger,
	})

	if err != nil {
		panic(fmt.Errorf("❌ MySQL[%s] connect failed: %w", env, err))
	}

	// 设置底层连接池参数（强烈建议加上）
	sqlDB, _ := engine.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(200)
	sqlDB.SetConnMaxLifetime(90 * time.Minute)

	engines.Store(env, engine)

	fmt.Printf("✅ MySQL[%s] connected -> %s\n", env, extractHost(dsn))
}

// 截取 MySQL DSN 的 host:port
func extractHost(dsn string) string {
	start := strings.Index(dsn, "(")
	end := strings.Index(dsn, ")")
	if start < 0 || end < 0 || end <= start {
		return "unknown-host"
	}
	return dsn[start+1 : end]
}

// 默认日志文件
func defaultLogFile(env string) string {
	if env == constant.Prod {
		return config.MysqlConfig.Prod.LogFile
	}
	return config.MysqlConfig.Test.LogFile
}

// 默认 DSN
func defaultDsn(env string) string {
	if env == constant.Prod {
		return config.MysqlConfig.Prod.Dsn
	}
	return config.MysqlConfig.Test.Dsn
}

// 获取 ORM（安全）
func GetOrm(env string) *gorm.DB {
	if engine, ok := engines.Load(env); ok {
		return engine.(*gorm.DB)
	}

	// fallback：test 环境
	if engine, ok := engines.Load(constant.Test); ok {
		return engine.(*gorm.DB)
	}

	panic("❌ no database initialized")
}

// 获取当前数据库名称
func GetCurrentDatabase(db *gorm.DB) (string, error) {
	var dbName string

	err := db.Raw("SELECT DATABASE()").Scan(&dbName).Error
	if err != nil {
		return "", err
	}
	return dbName, nil
}
