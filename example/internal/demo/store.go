package demo

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	redis8 "github.com/go-redis/redis/v8"
	"github.com/gogf/gcache-adapter/adapter"
	"github.com/gogf/gf/database/gredis"
	"github.com/gogf/gf/frame/g"
	redis9 "github.com/redis/go-redis/v9"
	"github.com/rockefeller-noah/jwt"
	"github.com/rockefeller-noah/jwt/adapter/gozero"
	"github.com/rockefeller-noah/jwt/adapter/redisv8"
	"github.com/rockefeller-noah/jwt/adapter/redisv9"
	zeroredis "github.com/zeromicro/go-zero/core/stores/redis"
)

// StoreConfig selects an example adapter and its connection settings.
type StoreConfig struct {
	Driver, Addrs, Username, Password string
	DB                                int
	Cluster                           bool
}

// OpenStore creates and checks a store. The returned cleanup must be called.
// "none" disables stateful token verification; it is not an in-memory store.
func OpenStore(ctx context.Context, cfg StoreConfig) (jwt.Store, func(), error) {
	noop := func() {}
	if cfg.Driver == "none" {
		return nil, noop, nil
	}
	var addrs []string
	for _, addr := range strings.Split(cfg.Addrs, ",") {
		addr = strings.TrimSpace(addr)
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return nil, noop, fmt.Errorf("invalid Redis address %q: %w", addr, err)
		}
		addrs = append(addrs, addr)
	}
	if cfg.DB < 0 || (cfg.Cluster && cfg.DB != 0) {
		return nil, noop, fmt.Errorf("Redis DB must be nonnegative; clusters require DB 0")
	}
	if !cfg.Cluster && len(addrs) != 1 {
		return nil, noop, fmt.Errorf("single-node mode requires exactly one Redis address")
	}
	switch cfg.Driver {
	case "redisv8":
		var client redis8.UniversalClient
		if cfg.Cluster {
			client = redis8.NewClusterClient(&redis8.ClusterOptions{
				Addrs: addrs, Username: cfg.Username, Password: cfg.Password,
			})
		} else {
			client = redis8.NewClient(&redis8.Options{
				Addr: addrs[0], Username: cfg.Username, Password: cfg.Password, DB: cfg.DB,
			})
		}
		cleanup := func() { _ = client.Close() }
		if err := client.Ping(ctx).Err(); err != nil {
			cleanup()
			return nil, noop, err
		}
		return redisv8.New(client), cleanup, nil
	case "redisv9":
		var client redis9.UniversalClient
		if cfg.Cluster {
			client = redis9.NewClusterClient(&redis9.ClusterOptions{
				Addrs: addrs, Username: cfg.Username, Password: cfg.Password,
			})
		} else {
			client = redis9.NewClient(&redis9.Options{
				Addr: addrs[0], Username: cfg.Username, Password: cfg.Password, DB: cfg.DB,
			})
		}
		cleanup := func() { _ = client.Close() }
		if err := client.Ping(ctx).Err(); err != nil {
			cleanup()
			return nil, noop, err
		}
		return redisv9.New(client), cleanup, nil
	case "gozero":
		if cfg.DB != 0 {
			return nil, noop, fmt.Errorf("go-zero's Redis wrapper uses DB 0")
		}
		mode := "node"
		if cfg.Cluster {
			mode = "cluster"
		}
		client, err := zeroredis.NewRedis(zeroredis.RedisConf{
			Host: strings.Join(addrs, ","), Type: mode, User: cfg.Username,
			Pass: cfg.Password, NonBlock: true,
		})
		if err != nil {
			return nil, noop, err
		}
		if !client.PingCtx(ctx) {
			return nil, noop, fmt.Errorf("cannot connect to go-zero Redis")
		}
		// go-zero manages shared connection pools and exposes no Close method.
		return gozero.New(client), noop, nil
	case "goframe":
		if cfg.Cluster || cfg.Username != "" {
			return nil, noop, fmt.Errorf("this GoFrame v1 example supports single-node password authentication only")
		}
		host, port, _ := net.SplitHostPort(addrs[0])
		portNumber, err := strconv.Atoi(port)
		if err != nil {
			return nil, noop, fmt.Errorf("invalid Redis port: %w", err)
		}
		gredis.SetConfig(&gredis.Config{
			Host: host, Port: portNumber, Db: cfg.DB, Pass: cfg.Password,
			ConnectTimeout: 5 * time.Second,
		}, "jwt")
		client := g.Redis("jwt")
		cleanup := func() { _ = client.Close() }
		if _, err := client.Ctx(ctx).Do("PING"); err != nil {
			cleanup()
			return nil, noop, err
		}
		return adapter.NewRedis(client), cleanup, nil
	default:
		return nil, noop, fmt.Errorf("unknown store %q; choose redisv8, redisv9, gozero, goframe, or none", cfg.Driver)
	}
}
