// Package redisv9 adapts go-redis v9 clients to jwt.Store.
package redisv9

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rockefeller-noah/jwt"
	"github.com/rockefeller-noah/jwt/internal/redisstore"
)

// New wraps a single-node, cluster, or failover client. The caller retains
// ownership of the client and is responsible for closing it.
func New(client redis.UniversalClient) jwt.Store {
	return redisstore.New(
		func(ctx context.Context, key string) (string, error) {
			value, err := client.Get(ctx, key).Result()
			if errors.Is(err, redis.Nil) {
				return "", nil
			}
			return value, err
		},
		func(ctx context.Context, key, value string, duration time.Duration) error {
			return client.Set(ctx, key, value, duration).Err()
		},
		func(ctx context.Context, key string) (int64, error) {
			return client.Del(ctx, key).Result()
		},
	)
}
