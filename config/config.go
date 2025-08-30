package config

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	cfg *Settings
)

type Settings struct {
	Settings  configsetting
	callbacks []func()
}

type configsetting struct {
	Application *Application
	Server      *Server
	Api         *Api
	Mysql       *Mysql
	Redis       *Redis
}

func (e *Settings) runCallback() {
	for i := range e.callbacks {
		e.callbacks[i]()
	}
}

// 初始化
func (e *Settings) Init() {
	e.init()
}

func (e *Settings) init() {
	e.runCallback()
}

// setup
func Setup(filePath string, fs ...func()) {
	cfg = &Settings{
		Settings: configsetting{
			Application: ApplicationConfig,
			Server:      ServerConfig,
			Api:         ApiConfig,
			Mysql:       MysqlConfig,
			Redis:       RedisConfig,
		},
		callbacks: fs,
	}
	initialize(filePath)
	cfg.Init()
}

// 读取配置文件
func initialize(filePath string) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(filePath)

	//3、启用环境变量读取
	viper.AutomaticEnv()
	//读取config
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("read config error:%v", err))
	}
	viper.Unmarshal(&cfg.Settings)

	// 监听配置文件变更(只能新增或修改配置项)
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		viper.Unmarshal(&cfg.Settings)
	})
}
