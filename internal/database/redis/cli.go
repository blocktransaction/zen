package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCli struct {
	client *redis.Client
	env    string
	ctx    context.Context
}

func NewRedisCli(ctx context.Context, env string) *RedisCli {
	return &RedisCli{
		env:    env,
		client: RedisClient(env),
		ctx:    ctx,
	}
}

// 获取key
func (r *RedisCli) Get(key string) string {
	return r.client.Get(r.ctx, key).Val()
}

// 设置key
func (r *RedisCli) Set(key string, value interface{}, expireTime time.Duration) error {
	return r.client.Set(r.ctx, key, value, expireTime).Err()
}

// 删除key
func (r *RedisCli) Del(keys ...string) error {
	return r.client.Del(r.ctx, keys...).Err()
}

// hset
func (r *RedisCli) HSet(key string, values ...interface{}) error {
	return r.client.HSet(r.ctx, key, values...).Err()
}

// hget
func (r *RedisCli) HGet(key, field string) string {
	return r.client.HGet(r.ctx, key, field).Val()
}

// hdel
func (r *RedisCli) HDel(key, field string) error {
	return r.client.HDel(r.ctx, key, field).Err()
}

func (r *RedisCli) Expire(key string, expireTime time.Duration) error {
	return r.client.Expire(r.ctx, key, expireTime).Err()
}

func (r *RedisCli) TTL(key string) time.Duration {
	return r.client.TTL(r.ctx, key).Val()
}

// hincrby
func (r *RedisCli) HIncrBy(key, field string, increment int64) error {
	return r.client.HIncrBy(r.ctx, key, field, increment).Err()
}

func (r *RedisCli) SPop(key string) string {
	return r.client.SPop(r.ctx, key).Val()
}

func (r *RedisCli) SAdd(key string, members ...interface{}) error {
	return r.client.SAdd(r.ctx, key, members...).Err()
}

func (r *RedisCli) SCard(key string) int64 {
	return r.client.SCard(r.ctx, key).Val()
}

func (r *RedisCli) SRem(key string, members ...interface{}) error {
	return r.client.SRem(r.ctx, key, members...).Err()
}

func (r *RedisCli) SMembers(key string) []string {
	return r.client.SMembers(r.ctx, key).Val()
}

func (r *RedisCli) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

func (r *RedisCli) Incr(key string) (int64, error) {
	cmd := r.client.Incr(r.ctx, key)
	return cmd.Val(), cmd.Err()
}

func (r *RedisCli) Unlink(key ...string) (int64, error) {
	cmd := r.client.Unlink(r.ctx, key...)
	return cmd.Val(), cmd.Err()
}
