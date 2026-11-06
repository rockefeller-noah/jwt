# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

适用于 Go 的框架无关 JWT 库，提供 HTTP 辅助方法与基于 Redis 的会话校验。支持 HS256/384/512、RS256/384/512 和 ES256/384/512。

## 安装

需要 Go 1.25 或更新版本。库本身可不使用 Redis，Redis 存储示例需要可连接的 Redis 服务。

```sh
go get github.com/rockefeller-noah/jwt
```

## 快速开始

在你的 Go 应用中将以下程序保存为 `main.go`：

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

运行 `JWT_SECRET=local-example-secret-change-me go run main.go`，预期输出为 `user-1`。

## Redis 适配器

通过 `jwt.WithStore(...)` 接入已有客户端，只需导入对应适配器：

| 客户端 | 导入路径 | 使用方式 |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

v8、v9 的 `New` 通过 `redis.UniversalClient` 同时接受 `*redis.Client` 和 `*redis.ClusterClient`，也可接收哨兵客户端，不会创建新客户端或连接池。go-zero 直接传入已有的 `*redis.Redis`，例如 `gozero.New(svcCtx.Redis)`。原有 GoFrame 的 `adapter.NewRedis(g.Redis("jwt"))` 接入方式继续支持。

三个内置适配器在键不存在时返回空字符串且不报错。过期时间精确到毫秒，最小正数按 1ms 处理；零表示不过期，负数或 `nil` 值表示删除。`Remove` 返回删除数量，逐键删除以兼容集群，多个键的删除不保证原子性。客户端由调用方负责关闭。go-zero 通过单键 `EVAL` 原子设置值与过期时间，需要相应 Redis 权限。

启用存储后，相同签发方和主体的最新登录或刷新会替换旧令牌，注销会删除会话记录。不配置 `WithStore` 时令牌是无状态的，注销不能撤销令牌，旧令牌在过期前仍有效。会话存储时长取有效期与刷新期的较大值。

## 可运行的 HTTP 示例

从仓库根目录开始，先在 `127.0.0.1:6379` 启动本地 Redis，再运行[示例](example/server.go)：

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

切换模式前先停止当前服务。集群模式需要已有的 Redis Cluster，多个独立 Redis 实例不能代替集群。

| 模式 | 命令 |
| --- | --- |
| go-redis v8 · 单机 | `go run . -store redisv8` |
| go-redis v8 · 集群 | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · 单机 | `go run . -store redisv9` |
| go-redis v9 · 集群 | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · 单机 | `go run . -store gozero` |
| go-zero · 集群 | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · 单机 | `go run . -store goframe` |
| 无 Redis／无状态 | `go run . -store none` |

### 配置

| 参数／环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | HTTP 监听地址 |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | 以逗号分隔的 Redis 入口地址 |
| `-redis-db` | `0` | 数据库编号；集群和 go-zero 必须为 0 |
| `-cluster` | `false` | v8、v9、go-zero 的集群模式 |
| `JWT_SECRET` | 必填 | JWT 签名密钥 |
| `REDIS_USERNAME` | 空 | Redis ACL 用户名；支持 v8、v9、go-zero |
| `REDIS_PASSWORD` | 空 | Redis 密码；适用于所有 Redis 模式 |

GoFrame v1 示例继续使用 `g.Redis("jwt")` 与 `adapter.NewRedis(...)`，支持单机 Redis、密码认证和 `-redis-db`，不支持集群或 ACL 用户名。参见[存储初始化](example/internal/demo/store.go)。示例在启动时检查连接，并处理 Ctrl+C 退出。

### HTTP 调用流程

需要 `curl` 和 `jq`。示例固定账号为 `demo`／`demo-password`；接入应用时应替换为自己的用户认证逻辑。

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

启用 Redis 时，注销后的最后一次请求返回 `401`。刷新后应使用新令牌。`none` 模式下注销返回 `501`，不会撤销令牌。

| 接口 | 认证 | 用途 |
| --- | --- | --- |
| `GET /healthz` | — | 仅检查 HTTP 服务存活，不检查 Redis 就绪状态 |
| `POST /login` | — | 校验示例 JSON 账号密码并签发令牌 |
| `GET /profile` | `Bearer` | 校验令牌并返回载荷 |
| `PUT /refresh` | `Bearer` | 刷新仍有效的令牌 |
| `DELETE /logout` | `Bearer` | 撤销当前存储的会话 |

导入 [Postman 集合](example/jwt.postman.json)，通过集合变量 `base_url` 和 `token` 配置地址与令牌，登录和刷新后会自动保存新令牌。请在启用 Redis 的模式下运行该集合。

## API 说明

完整接口参见 [JWT 操作](jwt.go)、[配置选项](options.go)和 [HTTP 辅助方法](http.go)。

| API | 用途 |
| --- | --- |
| `GenerateToken(subject, payload...)` | 为字符串主体与可选自定义载荷签发令牌 |
| `ParseToken(token)` / `RefreshToken(token)` | 校验载荷／签发替换令牌 |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | 配置存储时删除会话记录 |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | 使用 `time.Hour` 等时间单位，避免裸整数。默认有效期／刷新期为 2h／1h，常规校验同时检查这两项时限 |
| `WithLookupLocations("header:Authorization")` | 令牌查找位置，以逗号分隔：`header`、`query`、`cookie`、`form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | 校验请求／读取请求上下文中缓存的载荷 |

可使用 `jwt.IsMissingToken`、`jwt.IsExpiredToken`、`jwt.IsInvalidToken`、`jwt.IsAuthElsewhere` 或 `errors.Is` 判断错误。示例对登录 JSON 格式错误返回 `400`，认证失败返回 `401`，存储故障返回 `503`。

## 测试

```sh
go test ./...
cd example
go test ./...
```

测试基于 miniredis，覆盖 v8、v9、go-zero 的单机与集群客户端，以及 GoFrame 和无状态 HTTP 模式，验证过期、刷新、注销及错误处理。这些是 Redis 模拟测试，不代替真实多节点集群的部署测试。

## 许可证与来源

[MIT](LICENSE)。项目派生自 [dobyte/jwt](https://github.com/dobyte/jwt)，保留原作者版权声明。
