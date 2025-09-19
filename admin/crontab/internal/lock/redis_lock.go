package lock

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisLock struct {
	rdb *redis.Client
	// safe delete lua
	delLua string
}

func NewRedisLock(rdb *redis.Client) *RedisLock {
	return &RedisLock{
		rdb:    rdb,
		delLua: `if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end`,
	}
}

// Acquire tries to set NX; returns true if acquired
func (l *RedisLock) Acquire(ctx context.Context, key, val string, ttl time.Duration) (bool, error) {
	return l.rdb.SetNX(ctx, key, val, ttl).Result()
}

// SafeRelease deletes lock only if value matches
func (l *RedisLock) SafeRelease(ctx context.Context, key, val string) error {
	res, err := l.rdb.Eval(ctx, l.delLua, []string{key}, val).Result()
	if err != nil {
		return err
	}
	if n, ok := res.(int64); ok && n == 0 {
		return errors.New("lock not held")
	}
	return nil
}

func (l *RedisLock) Extend(ctx context.Context, key, val string, ttl time.Duration) error {
	// 仅在锁被持有时续租
	lua := `
    if redis.call("get", KEYS[1]) == ARGV[1] then
        return redis.call("pexpire", KEYS[1], ARGV[2])
    else
        return 0
    end
    `
	res, err := l.rdb.Eval(ctx, lua, []string{key}, val, int(ttl.Milliseconds())).Result()
	if err != nil {
		return err
	}
	if n, ok := res.(int64); !ok || n == 0 {
		return errors.New("extend failed, lock not held")
	}
	return nil
}
