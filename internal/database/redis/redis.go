package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var clients = make(map[string]*redis.Client)

// 初始化所有 Redis 客户端
func Setup(logger *zap.Logger, logResult bool) {
	initRedis(constant.Prod, logger, logResult)
	initRedis(constant.Test, logger, logResult)
}

// 初始化单个 redis 客户端
func initRedis(env string, logger *zap.Logger, logResult bool) {
	// 处理默认值
	cfg := defaultConfig(env)
	addr := defaultString(cfg.Addr, "127.0.0.1:6379")
	username := cfg.UserName
	password := cfg.Password
	dialTimeout := defaultDuration(cfg.DialTimeout, 5*time.Second)
	poolSize := defaultInt(cfg.PoolSize, 10)

	cli := redis.NewClient(&redis.Options{
		Addr:        addr,
		Username:    username,
		Password:    password,
		DialTimeout: dialTimeout,
		PoolSize:    poolSize,
	})

	// ping 校验
	if err := cli.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Errorf("redis[%s] connect failed: %w", env, err))
	}
	cli.AddHook(NewRedisLogger(NewZapAdapter(logger), logResult))
	clients[env] = cli
	fmt.Printf("redis[%s] connected: %s\n", env, addr)
}

// 获取对应环境的 redis 客户端
func RedisClient(env string) *redis.Client {
	if cli, ok := clients[env]; ok {
		return cli
	}
	// 默认返回测试环境，避免 nil
	return clients[constant.Test]
}

// ---------- 默认值处理函数 ----------
func defaultString(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func defaultInt(val, def int) int {
	if val <= 0 {
		return def
	}
	return val
}

func defaultDuration(val int, def time.Duration) time.Duration {
	if val <= 0 {
		return def
	}
	return time.Duration(val) * time.Second
}

func defaultConfig(env string) *config.RedisDefaultConfig {
	if env == constant.Test {
		return &config.RedisConfig.Test
	}
	return &config.RedisConfig.Prod
}
