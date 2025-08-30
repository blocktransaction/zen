package config

type Redis struct {
	Prod struct {
		Addr        string
		UserName    string
		Password    string
		DialTimeout int
		PoolSize    int
	}
	Test struct {
		Addr        string
		UserName    string
		Password    string
		DialTimeout int
		PoolSize    int
	}
}

var RedisConfig = new(Redis)
