package mysql

import (
	"fmt"
	"strings"
	"time"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var engines = make(map[string]*gorm.DB)

// 初始化数据库连接
func Setup() {
	//生产环境
	initMysql(constant.Prod)
	//测试环境
	initMysql(constant.Test)
}

// 连接不同环境的mysql
func initMysql(env string) {
	logger := NewLogger(LogConfig{
		Rotate:        true, // 开启日志轮转
		LogFile:       defaultLogFile(env),
		EnableMasking: true, // 开启脱敏
		Config: logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      logger.Info,
		},
	})

	dsn := defaultDsn(env)
	engine, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt:            true,   // 启用预编译语句
		SkipDefaultTransaction: true,   // 禁用默认事务
		Logger:                 logger, // 日志配置
	})

	if err != nil {
		panic(fmt.Errorf("mysql[%s] connect failed: %w", env, err))
	}
	engines[env] = engine
	fmt.Printf("mysql[%s] connected: %s\n", env, cutDsn(dsn))
}

// 截取mysql [server:port]
func cutDsn(dsn string) string {
	start := strings.Index(dsn, "(") + 1
	end := strings.Index(dsn, ")")
	return dsn[start:end]
}

// 默认配置文件（默认：测试）
func defaultLogFile(env string) string {
	if env == constant.Prod {
		return config.MysqlConfig.Prod.LogFile
	}
	return config.MysqlConfig.Test.LogFile
}

// 默认连接串（默认：测试）
func defaultDsn(env string) string {
	if env == constant.Prod {
		return config.MysqlConfig.Prod.Dsn
	}
	return config.MysqlConfig.Test.Dsn
}

// 获取数据库连接
func GetOrm(env string) *gorm.DB {
	if engine, ok := engines[env]; ok {
		return engine
	}
	return engines[config.MysqlConfig.Test.Dsn]
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
