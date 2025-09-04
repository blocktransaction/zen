package config

type Redis struct {
	Prod RedisDefaultConfig
	Test RedisDefaultConfig
}
type RedisDefaultConfig struct {
	Addr        string
	UserName    string
	Password    string
	DialTimeout int
	PoolSize    int
}

var RedisConfig = new(Redis)
