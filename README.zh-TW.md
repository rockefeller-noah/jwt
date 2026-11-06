# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

適用於 Go 的框架無關 JWT 函式庫，提供 HTTP 輔助方法與基於 Redis 的工作階段驗證。支援 HS256/384/512、RS256/384/512 和 ES256/384/512。

## 安裝

需要 Go 1.25 或更新版本。函式庫本身可不使用 Redis，Redis 儲存範例需要可連線的 Redis 服務。

```sh
go get github.com/rockefeller-noah/jwt
```

## 快速開始

在你的 Go 應用程式中將以下程式儲存為 `main.go`：

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

執行 `JWT_SECRET=local-example-secret-change-me go run main.go`，預期輸出為 `user-1`。

## Redis 轉接器

透過 `jwt.WithStore(...)` 接入現有用戶端，只需匯入對應轉接器：

| 用戶端 | 匯入路徑 | 使用方式 |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

v8、v9 的 `New` 透過 `redis.UniversalClient` 同時接受 `*redis.Client` 和 `*redis.ClusterClient`，也可接收哨兵用戶端，不會建立新用戶端或連線池。go-zero 直接傳入現有的 `*redis.Redis`，例如 `gozero.New(svcCtx.Redis)`。原有 GoFrame 的 `adapter.NewRedis(g.Redis("jwt"))` 接入方式繼續支援。

三個內建轉接器在鍵不存在時回傳空字串且不報錯。到期時間精確到毫秒，最小正數按 1ms 處理；零表示不會到期，負數或 `nil` 值表示刪除。`Remove` 回傳刪除數量，逐鍵刪除以相容叢集，多個鍵的刪除不保證原子性。用戶端由呼叫端負責關閉。go-zero 透過單鍵 `EVAL` 以原子方式設定值與到期時間，需要相應 Redis 權限。

啟用儲存後，相同簽發者與主體的最新登入或更新會替換舊權杖，登出會刪除工作階段紀錄。不設定 `WithStore` 時權杖為無狀態，登出無法撤銷權杖，舊權杖在到期前仍有效。工作階段儲存時間取有效期限與更新期限的較大值。

## 可執行的 HTTP 範例

從儲存庫根目錄開始，先在 `127.0.0.1:6379` 啟動本機 Redis，再執行[範例](example/server.go)：

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

切換模式前請先停止目前的服務。叢集模式需要現有的 Redis Cluster，多個獨立 Redis 執行個體不能取代叢集。

| 模式 | 指令 |
| --- | --- |
| go-redis v8 · 單機 | `go run . -store redisv8` |
| go-redis v8 · 叢集 | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · 單機 | `go run . -store redisv9` |
| go-redis v9 · 叢集 | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · 單機 | `go run . -store gozero` |
| go-zero · 叢集 | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · 單機 | `go run . -store goframe` |
| 無 Redis／無狀態 | `go run . -store none` |

### 設定

| 參數／環境變數 | 預設值 | 說明 |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | HTTP 監聽位址 |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | 以逗號分隔的 Redis 入口位址 |
| `-redis-db` | `0` | 資料庫編號；叢集和 go-zero 必須為 0 |
| `-cluster` | `false` | v8、v9、go-zero 的叢集模式 |
| `JWT_SECRET` | 必填 | JWT 簽章金鑰 |
| `REDIS_USERNAME` | 空 | Redis ACL 使用者名稱；支援 v8、v9、go-zero |
| `REDIS_PASSWORD` | 空 | Redis 密碼；適用於所有 Redis 模式 |

GoFrame v1 範例繼續使用 `g.Redis("jwt")` 與 `adapter.NewRedis(...)`，支援單機 Redis、密碼驗證和 `-redis-db`，不支援叢集或 ACL 使用者名稱。參見[儲存初始化](example/internal/demo/store.go)。範例會在啟動時檢查連線，並處理 Ctrl+C 結束。

### HTTP 呼叫流程

需要 `curl` 和 `jq`。範例固定帳號為 `demo`／`demo-password`；接入應用程式時應替換為自己的使用者驗證邏輯。

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

啟用 Redis 時，登出後的最後一次請求回傳 `401`。更新後應使用新權杖。`none` 模式下登出回傳 `501`，不會撤銷權杖。

| 端點 | 驗證 | 用途 |
| --- | --- | --- |
| `GET /healthz` | — | 僅檢查 HTTP 服務是否存活，不檢查 Redis 是否就緒 |
| `POST /login` | — | 驗證範例 JSON 帳號密碼並簽發權杖 |
| `GET /profile` | `Bearer` | 驗證權杖並回傳承載資料 |
| `PUT /refresh` | `Bearer` | 更新仍有效的權杖 |
| `DELETE /logout` | `Bearer` | 撤銷目前儲存的工作階段 |

匯入 [Postman 集合](example/jwt.postman.json)，透過集合變數 `base_url` 和 `token` 設定位址與權杖，登入和更新後會自動儲存新權杖。請在啟用 Redis 的模式下執行此集合。

## API 說明

完整介面參見 [JWT 操作](jwt.go)、[設定選項](options.go)和 [HTTP 輔助方法](http.go)。

| API | 用途 |
| --- | --- |
| `GenerateToken(subject, payload...)` | 為字串主體與選用的自訂承載資料簽發權杖 |
| `ParseToken(token)` / `RefreshToken(token)` | 驗證承載資料／簽發替換權杖 |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | 設定儲存時刪除工作階段紀錄 |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | 使用 `time.Hour` 等時間單位，避免單純整數。預設有效期限／更新期限為 2h／1h，一般驗證會同時檢查這兩項時限 |
| `WithLookupLocations("header:Authorization")` | 權杖查找位置，以逗號分隔：`header`、`query`、`cookie`、`form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | 驗證請求／讀取請求上下文中快取的承載資料 |

可使用 `jwt.IsMissingToken`、`jwt.IsExpiredToken`、`jwt.IsInvalidToken`、`jwt.IsAuthElsewhere` 或 `errors.Is` 判斷錯誤。範例對登入 JSON 格式錯誤回傳 `400`，驗證失敗回傳 `401`，儲存故障回傳 `503`。

## 測試

```sh
go test ./...
cd example
go test ./...
```

測試基於 miniredis，涵蓋 v8、v9、go-zero 的單機與叢集用戶端，以及 GoFrame 和無狀態 HTTP 模式，驗證到期、更新、登出及錯誤處理。這些是 Redis 模擬測試，不能取代真實多節點叢集的部署測試。

## 授權條款與來源

[MIT](LICENSE)。專案衍生自 [dobyte/jwt](https://github.com/dobyte/jwt)，保留原作者著作權聲明。
