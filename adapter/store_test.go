package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2/server"
)

func TestStore(t *testing.T) {
	for _, name := range clientNames {
		t.Run(name, func(t *testing.T) {
			s, store := newTestStore(t, name)
			ctx := context.Background()
			if value, err := store.Get(ctx, "missing"); err != nil || value != "" {
				t.Fatalf("missing key: value=%v, err=%v", value, err)
			}
			if err := store.Set(ctx, 42, []byte("uuid"), 1500*time.Millisecond); err != nil {
				t.Fatal(err)
			}
			if value, err := store.Get(ctx, []byte("42")); err != nil || value != "uuid" {
				t.Fatalf("stored value: value=%v, err=%v", value, err)
			}
			if ttl := s.TTL("42"); ttl != 1500*time.Millisecond {
				t.Fatalf("TTL=%v, want 1.5s", ttl)
			}
			if err := store.Set(ctx, 42, "persistent", 0); err != nil {
				t.Fatal(err)
			}
			if ttl := s.TTL("42"); ttl != 0 {
				t.Fatalf("persistent TTL=%v", ttl)
			}
			if err := store.Set(ctx, "short", "value", time.Microsecond); err != nil {
				t.Fatal(err)
			}
			if ttl := s.TTL("short"); ttl != time.Millisecond {
				t.Fatalf("short TTL=%v, want 1ms", ttl)
			}
			s.FastForward(2 * time.Millisecond)
			if value, err := store.Get(ctx, "short"); err != nil || value != "" {
				t.Fatalf("expired value=%v, err=%v", value, err)
			}
			for _, deletion := range []struct {
				value    any
				duration time.Duration
			}{{nil, time.Hour}, {"ignored", -time.Second}} {
				if err := store.Set(ctx, "delete", "value", 0); err != nil {
					t.Fatal(err)
				}
				if err := store.Set(ctx, "delete", deletion.value, deletion.duration); err != nil {
					t.Fatal(err)
				}
				if s.Exists("delete") {
					t.Fatal("Set should delete the key")
				}
			}
			for _, key := range []string{"{first}:key", "{second}:key"} {
				if err := store.Set(ctx, key, "value", time.Hour); err != nil {
					t.Fatal(err)
				}
			}
			count, err := store.Remove(ctx, "{first}:key", []byte("{second}:key"), "missing")
			if err != nil || count != int64(2) {
				t.Fatalf("Remove count=%v, err=%v", count, err)
			}
			if count, err := store.Remove(ctx); err != nil || count != int64(0) {
				t.Fatalf("empty Remove count=%v, err=%v", count, err)
			}
		})
	}
}

func TestStoreErrors(t *testing.T) {
	for _, name := range clientNames {
		t.Run(name, func(t *testing.T) {
			s, store := newTestStore(t, name)
			ctx := context.Background()
			if _, err := s.Push("wrong-type", "value"); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Get(ctx, "wrong-type"); err == nil {
				t.Fatal("Get must propagate Redis errors")
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := store.Get(cancelled, "key"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled Get: %v", err)
			}
			if err := store.Set(cancelled, "key", "value", time.Second); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled Set: %v", err)
			}
			if _, err := store.Remove(cancelled, "key"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled Remove: %v", err)
			}
			if err := s.Set("first", "value"); err != nil {
				t.Fatal(err)
			}
			s.Server().SetPreHook(func(c *server.Peer, cmd string, args ...string) bool {
				if cmd == "DEL" && len(args) == 1 && args[0] == "failed" {
					c.WriteError("ERR deletion failed")
					return true
				}
				return false
			})
			if count, err := store.Remove(ctx, "first", "failed"); err == nil || count != int64(1) {
				t.Fatalf("partial Remove count=%v, err=%v", count, err)
			}
		})
	}
}
