package bunx

import (
	"database/sql"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
)

var engines = make(map[string]*bun.DB)

func Setup() {
	//生产
	initMysql(constant.Prod)
	//测试
	initMysql(constant.Test)
}

// 初始化connect
func initMysql(env string) {
	// Open database
	sqldb, err := sql.Open("mysql", defaultDsn(env))
	if err != nil {
		panic(err)
	}

	// Create Bun instance
	engines[env] = bun.NewDB(sqldb, mysqldialect.New()) // 设置自定义日志级别
	// engines[env].Set(&BunLogger{
	// 	Logger: log.New(os.Stdout, "[BUN] ", log.LstdFlags|log.Lshortfile),
	// })

}

// 默认连接串（默认：测试）
func defaultDsn(env string) string {
	if env == constant.Prod {
		return config.MysqlConfig.Prod.Dsn
	}
	return config.MysqlConfig.Test.Dsn
}
