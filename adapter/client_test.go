package adapter_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	redis8 "github.com/go-redis/redis/v8"
	redis9 "github.com/redis/go-redis/v9"
	"github.com/rockefeller-noah/jwt"
	"github.com/rockefeller-noah/jwt/adapter/gozero"
	"github.com/rockefeller-noah/jwt/adapter/redisv8"
	"github.com/rockefeller-noah/jwt/adapter/redisv9"
	zeroredis "github.com/zeromicro/go-zero/core/stores/redis"
)

var clientNames = []string{
	"v8/node", "v8/cluster", "v9/node", "v9/cluster", "gozero/node", "gozero/cluster",
}

func newTestStore(t *testing.T, name string) (*miniredis.Miniredis, jwt.Store) {
	t.Helper()
	s := miniredis.RunT(t)
	// Reject multi-key DEL so even a one-node test cluster catches unsafe
	// cross-slot deletion commands sent by an adapter.
	s.Server().SetPreHook(func(c *server.Peer, cmd string, args ...string) bool {
		if cmd == "DEL" && len(args) > 1 {
			c.WriteError("CROSSSLOT Keys in request don't hash to the same slot")
			return true
		}
		return false
	})
	switch name {
	case "v8/node":
		client := redis8.NewClient(&redis8.Options{Addr: s.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		return s, redisv8.New(client)
	case "v8/cluster":
		client := redis8.NewClusterClient(&redis8.ClusterOptions{Addrs: []string{s.Addr()}})
		t.Cleanup(func() { _ = client.Close() })
		return s, redisv8.New(client)
	case "v9/node":
		client := redis9.NewClient(&redis9.Options{Addr: s.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		return s, redisv9.New(client)
	case "v9/cluster":
		client := redis9.NewClusterClient(&redis9.ClusterOptions{Addrs: []string{s.Addr()}})
		t.Cleanup(func() { _ = client.Close() })
		return s, redisv9.New(client)
	case "gozero/node", "gozero/cluster":
		mode := "node"
		if name == "gozero/cluster" {
			mode = "cluster"
		}
		client, err := zeroredis.NewRedis(zeroredis.RedisConf{Host: s.Addr(), Type: mode})
		if err != nil {
			t.Fatal(err)
		}
		return s, gozero.New(client)
	default:
		t.Fatalf("unknown client: %s", name)
		return nil, nil
	}
}
