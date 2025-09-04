package mysql

import (
	"fmt"
	"time"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbengine, dbengineTest *gorm.DB
)

// 初始化数据库连接
func Setup() {
	if err := initialize(); err != nil {
		panic(fmt.Sprintf("mysql prod environment init failed: %v", err))
	}
	if err := initializeTest(); err != nil {
		panic(fmt.Sprintf("mysql test environment init failed: %v", err))
	}
}

// 生产环境初始化
func initialize() (err error) {
	logger := NewLogger(LogConfig{
		LogFile: config.MysqlConfig.Prod.LogFile,
		Rotate:  true, // 开启日志轮转
		Config: logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
		},
	})

	dbengine, err = gorm.Open(mysql.Open(config.MysqlConfig.Prod.Dsn), &gorm.Config{
		PrepareStmt:            true,   // 启用预编译语句
		SkipDefaultTransaction: true,   // 禁用默认事务
		Logger:                 logger, // 日志配置
	})

	return
}

// 测试环境初始化
func initializeTest() (err error) {
	logger := NewLogger(LogConfig{
		LogFile: config.MysqlConfig.Test.LogFile,
		Rotate:  true, // 开启日志轮转
		Config: logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
		},
	})

	dbengineTest, err = gorm.Open(mysql.Open(config.MysqlConfig.Test.Dsn), &gorm.Config{
		PrepareStmt:            true,   // 启用预编译语句
		SkipDefaultTransaction: true,   // 禁用默认事务
		Logger:                 logger, // 日志配置
	})
	return
}

// 获取数据库连接
func GetOrm(env string) *gorm.DB {
	if env == constant.Prod {
		return dbengine
	}
	return dbengineTest
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
