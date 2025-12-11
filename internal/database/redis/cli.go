package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCli struct {
	client *redis.Client
	env    string
	ctx    context.Context
}

func NewRedisCli(ctx context.Context, env string) *RedisCli {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RedisCli{
		env:    env,
		client: RedisClient(env),
		ctx:    ctx,
	}
}

func (r *RedisCli) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

func (r *RedisCli) Set(key string, value interface{}, expire time.Duration) error {
	return r.client.Set(r.ctx, key, value, expire).Err()
}

func (r *RedisCli) Del(keys ...string) (int64, error) {
	return r.client.Del(r.ctx, keys...).Result()
}

func (r *RedisCli) Expire(key string, expire time.Duration) error {
	return r.client.Expire(r.ctx, key, expire).Err()
}

func (r *RedisCli) TTL(key string) (time.Duration, error) {
	return r.client.TTL(r.ctx, key).Result()
}

func (r *RedisCli) HSet(key string, field string, value interface{}) error {
	return r.client.HSet(r.ctx, key, field, value).Err()
}

func (r *RedisCli) HGet(key string, field string) (string, error) {
	return r.client.HGet(r.ctx, key, field).Result()
}

func (r *RedisCli) HDel(key string, fields ...string) (int64, error) {
	return r.client.HDel(r.ctx, key, fields...).Result()
}

func (r *RedisCli) HIncrBy(key, field string, incr int64) (int64, error) {
	return r.client.HIncrBy(r.ctx, key, field, incr).Result()
}

func (r *RedisCli) SAdd(key string, members ...interface{}) (int64, error) {
	return r.client.SAdd(r.ctx, key, members...).Result()
}

func (r *RedisCli) SRem(key string, members ...interface{}) (int64, error) {
	return r.client.SRem(r.ctx, key, members...).Result()
}

func (r *RedisCli) SPop(key string) (string, error) {
	return r.client.SPop(r.ctx, key).Result()
}

func (r *RedisCli) SCard(key string) (int64, error) {
	return r.client.SCard(r.ctx, key).Result()
}

func (r *RedisCli) SMembers(key string) ([]string, error) {
	return r.client.SMembers(r.ctx, key).Result()
}

func (r *RedisCli) Incr(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

func (r *RedisCli) Unlink(keys ...string) (int64, error) {
	return r.client.Unlink(r.ctx, keys...).Result()
}

func (r *RedisCli) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

func (r *RedisCli) HGetAllFast(key string) (map[string]string, error) {
	cmd := r.client.Do(r.ctx, "HGETALL", key)
	raw, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	// raw 类型是 []interface{}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, errors.New("invalid HGETALL resp")
	}

	// 分配 map（容量 = pair 数）
	m := make(map[string]string, len(arr)/2)

	// 手动解析 pair：key=value
	for i := 0; i < len(arr); i += 2 {
		k, ok1 := arr[i].(string)
		v, ok2 := arr[i+1].(string)
		if !ok1 || !ok2 {
			return nil, errors.New("invalid HGETALL element type")
		}
		m[k] = v
	}

	return m, nil
}
