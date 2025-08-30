package config

type Mysql struct {
	Prod struct {
		Dsn     string
		LogFile string
	}
	Test struct {
		Dsn     string
		LogFile string
	}
}

var MysqlConfig = new(Mysql)
