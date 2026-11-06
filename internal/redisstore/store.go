// Package redisstore provides shared behavior for the Redis store adapters.
package redisstore

import (
	"context"
	"time"

	"github.com/rockefeller-noah/jwt"
	"github.com/rockefeller-noah/jwt/internal/conv"
)

// Store implements jwt.Store using single-key Redis operations.
type Store struct {
	get    func(context.Context, string) (string, error)
	set    func(context.Context, string, string, time.Duration) error
	remove func(context.Context, string) (int64, error)
}

var _ jwt.Store = (*Store)(nil)

// New wraps client-specific Redis commands. Missing keys must return "", nil.
func New(
	get func(context.Context, string) (string, error),
	set func(context.Context, string, string, time.Duration) error,
	remove func(context.Context, string) (int64, error),
) *Store {
	return &Store{get: get, set: set, remove: remove}
}

// Get returns a string value, or an empty string if the key does not exist.
func (s *Store) Get(ctx context.Context, key any) (any, error) {
	return s.get(ctx, conv.String(key))
}

// Set writes a value atomically with its expiration. Zero duration persists;
// negative duration or nil value deletes the key. Positive durations shorter
// than a millisecond are rounded up to one millisecond.
func (s *Store) Set(ctx context.Context, key, value any, duration time.Duration) error {
	k := conv.String(key)
	if value == nil || duration < 0 {
		_, err := s.remove(ctx, k)
		return err
	}
	if duration > 0 && duration < time.Millisecond {
		duration = time.Millisecond
	}
	return s.set(ctx, k, conv.String(value), duration)
}

// Remove deletes keys individually so different cluster slots are supported.
// It returns the number of deleted keys, including partial progress on error.
// An empty key list is a no-op. Multiple deletions are not atomic.
func (s *Store) Remove(ctx context.Context, keys ...any) (any, error) {
	var removed int64
	for _, key := range keys {
		n, err := s.remove(ctx, conv.String(key))
		removed += n
		if err != nil {
			return removed, err
		}
	}
	return removed, nil
}
