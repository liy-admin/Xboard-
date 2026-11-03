package jobs

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisAdapter 把 go-redis 客户端适配为 RedisClient。
func NewRedisAdapter(c *redis.Client) RedisClient { return redisAdapter{c} }

type redisAdapter struct{ c *redis.Client }

func (r redisAdapter) SCard(ctx context.Context, key string) (int64, error) {
	return r.c.SCard(ctx, key).Result()
}

func (r redisAdapter) SPopN(ctx context.Context, key string, n int64) ([]string, error) {
	return r.c.SPopN(ctx, key, n).Result()
}

func (r redisAdapter) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return r.c.SetNX(ctx, key, value, ttl).Result()
}

func (r redisAdapter) Del(ctx context.Context, keys ...string) error {
	return r.c.Del(ctx, keys...).Err()
}

func (r redisAdapter) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := r.c.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	return v, err == nil, err
}

func (r redisAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return r.c.Set(ctx, key, value, ttl).Err()
}
