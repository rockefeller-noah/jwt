// Package gozero adapts go-zero's redis.Redis to jwt.Store.
package gozero

import (
	"context"
	"time"

	"github.com/rockefeller-noah/jwt"
	"github.com/rockefeller-noah/jwt/internal/redisstore"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// A single-key script preserves millisecond TTLs and atomically sets the value
// and its expiration, including when go-zero is configured for Redis Cluster.
const setWithExpiration = `return redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])`

// New wraps an existing Redis instance configured for either node or cluster
// mode. The caller retains ownership of the client.
func New(client *redis.Redis) jwt.Store {
	return redisstore.New(
		client.GetCtx,
		func(ctx context.Context, key, value string, duration time.Duration) error {
			if duration == 0 {
				return client.SetCtx(ctx, key, value)
			}
			_, err := client.EvalCtx(ctx, setWithExpiration,
				[]string{key}, value, duration.Milliseconds())
			return err
		},
		func(ctx context.Context, key string) (int64, error) {
			n, err := client.DelCtx(ctx, key)
			return int64(n), err
		},
	)
}
