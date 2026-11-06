# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

A framework-independent JWT library for Go, with HTTP helpers and Redis-backed session verification. Supports HS256/384/512, RS256/384/512, and ES256/384/512.

## Installation

Requires Go 1.25 or later. Redis is optional for the library and required for the Redis-backed examples.

```sh
go get github.com/rockefeller-noah/jwt
```

## Quick start

Save this program as `main.go` in your Go application:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rockefeller-noah/jwt"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	auth, err := jwt.NewJWT(
		jwt.WithIssuer("my-service"),
		jwt.WithSecretKey(secret),
		jwt.WithValidDuration(time.Hour),
		jwt.WithRefreshDuration(time.Hour),
	)
	if err != nil {
		log.Fatal(err)
	}
	token, err := auth.GenerateToken("user-1", jwt.Payload{"role": "admin"})
	if err != nil {
		log.Fatal(err)
	}
	claims, err := auth.ParseToken(token.Token)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(claims.Subject())
}
```

Run `JWT_SECRET=local-example-secret-change-me go run main.go`. Expected output: `user-1`.

## Redis adapters

Pass your existing client to `jwt.WithStore(...)`. Import only the adapter you need:

| Client | Import path | Usage |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

For v8 and v9, `New` accepts `*redis.Client` or `*redis.ClusterClient` through `redis.UniversalClient`; Sentinel clients are also accepted. No new client or pool is created. For go-zero, pass an existing `*redis.Redis`, such as `gozero.New(svcCtx.Redis)`. GoFrame's original `adapter.NewRedis(g.Redis("jwt"))` integration remains supported.

The three built-in adapters return an empty string without an error for missing keys. Expiration uses millisecond precision, with a minimum positive TTL of 1ms; zero persists, and negative durations or `nil` values delete. `Remove` returns a deletion count and deletes keys individually for cluster compatibility; multiple deletions are not atomic. The caller owns client cleanup. go-zero uses single-key `EVAL` for atomic writes with TTL and requires that Redis permission.

With a store, the latest login or refresh replaces the previous token for the same issuer and subject; logout removes its session record. Without `WithStore`, tokens are stateless: logout cannot revoke them and previous tokens remain valid until expiration. Session TTL is the larger of the valid and refresh durations.

## Runnable HTTP example

From the repository root, start a local Redis server at `127.0.0.1:6379`, then run [the example](example/server.go):

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

Stop the current server before switching modes. Cluster modes require an existing Redis Cluster, not several independent Redis servers.

| Mode | Command |
| --- | --- |
| go-redis v8 · single node | `go run . -store redisv8` |
| go-redis v8 · cluster | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · single node | `go run . -store redisv9` |
| go-redis v9 · cluster | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · single node | `go run . -store gozero` |
| go-zero · cluster | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · single node | `go run . -store goframe` |
| No Redis / stateless | `go run . -store none` |

### Configuration

| Flag / environment variable | Default | Meaning |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | HTTP listen address |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | Comma-separated Redis seed addresses |
| `-redis-db` | `0` | Database index; clusters and go-zero require 0 |
| `-cluster` | `false` | Cluster mode for v8, v9, and go-zero |
| `JWT_SECRET` | required | JWT signing secret |
| `REDIS_USERNAME` | empty | Redis ACL username; v8, v9, and go-zero |
| `REDIS_PASSWORD` | empty | Redis password; all Redis modes |

The GoFrame v1 example uses `g.Redis("jwt")` with `adapter.NewRedis(...)`; it supports single-node Redis, password authentication, and `-redis-db`. It does not support cluster mode or ACL usernames. See [store initialization](example/internal/demo/store.go). The demo checks connectivity on startup and handles Ctrl+C shutdown.

### HTTP walkthrough

Requires `curl` and `jq`. The demo-only account is `demo` / `demo-password`; replace this fixed check with your own user authentication in an application.

```sh
TOKEN=$(curl -fsS -X POST http://127.0.0.1:8888/login \
  -H 'Content-Type: application/json' \
  -d '{"account":"demo","password":"demo-password"}' | jq -r '.data.token')
curl -fsS http://127.0.0.1:8888/profile -H "Authorization: Bearer $TOKEN"
TOKEN=$(curl -fsS -X PUT http://127.0.0.1:8888/refresh \
  -H "Authorization: Bearer $TOKEN" | jq -r '.data.token')
curl -fsS -X DELETE http://127.0.0.1:8888/logout -H "Authorization: Bearer $TOKEN"
curl -i http://127.0.0.1:8888/profile -H "Authorization: Bearer $TOKEN"
```

With Redis enabled, the last request returns `401` after logout. Always use the token returned by refresh. In `none` mode, logout returns `501` and does not revoke tokens.

| Endpoint | Authentication | Purpose |
| --- | --- | --- |
| `GET /healthz` | — | HTTP liveness only; not a Redis readiness probe |
| `POST /login` | — | Validate the demo JSON credentials and issue a token |
| `GET /profile` | `Bearer` | Validate the token and return its claims |
| `PUT /refresh` | `Bearer` | Refresh a still-valid token |
| `DELETE /logout` | `Bearer` | Revoke the current stored session |

Import [the Postman collection](example/jwt.postman.json). It uses `base_url` and `token` collection variables and automatically saves login/refresh tokens. Run it with a Redis-backed mode.

## API notes

See [JWT operations](jwt.go), [options](options.go), and [HTTP helpers](http.go) for the full API.

| API | Purpose |
| --- | --- |
| `GenerateToken(subject, payload...)` | Issue a token for a string subject and optional custom claims |
| `ParseToken(token)` / `RefreshToken(token)` | Validate claims / issue a replacement token |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | Remove the session record when a store is configured |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | Use `time.Hour`, not bare integers. Defaults: 2h / 1h; normal validation checks both time limits |
| `WithLookupLocations("header:Authorization")` | Token lookup locations, comma-separated: `header`, `query`, `cookie`, `form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | Validate a request / read claims cached in its context |

Use `jwt.IsMissingToken`, `jwt.IsExpiredToken`, `jwt.IsInvalidToken`, `jwt.IsAuthElsewhere`, or `errors.Is` to classify errors. The example returns `400` for malformed login JSON, `401` for authentication failures, and `503` for store failures.

## Tests

```sh
go test ./...
cd example
go test ./...
```

Tests use miniredis to exercise v8, v9, and go-zero single-node/cluster clients, plus the GoFrame and stateless HTTP modes. They cover expiration, refresh, revocation, and errors. These are simulated Redis tests, not a deployment test against a multi-node cluster.

## License and origin

[MIT](LICENSE). Forked from [dobyte/jwt](https://github.com/dobyte/jwt); original copyright notices are retained.
