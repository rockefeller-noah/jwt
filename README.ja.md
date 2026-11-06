# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

Go 向けのフレームワークに依存しない JWT ライブラリです。HTTP ヘルパーと Redis によるセッション検証を提供し、HS256/384/512、RS256/384/512、ES256/384/512 に対応しています。

## インストール

Go 1.25 以降が必要です。ライブラリ自体では Redis は任意ですが、Redis を使用するサンプルには接続可能な Redis サーバーが必要です。

```sh
go get github.com/rockefeller-noah/jwt
```

## クイックスタート

Go アプリケーション内で次のプログラムを `main.go` として保存してください。

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

`JWT_SECRET=local-example-secret-change-me go run main.go` を実行すると、`user-1` が出力されます。

## Redis アダプター

`jwt.WithStore(...)` に既存のクライアントを渡します。使用するアダプターだけをインポートしてください。

| クライアント | インポートパス | 使用方法 |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

v8 と v9 の `New` は `redis.UniversalClient` を通じて `*redis.Client` と `*redis.ClusterClient` を受け取り、Sentinel クライアントにも対応します。新しいクライアントや接続プールは作成しません。go-zero では `gozero.New(svcCtx.Redis)` のように既存の `*redis.Redis` を渡します。従来の GoFrame の `adapter.NewRedis(g.Redis("jwt"))` も引き続き利用できます。

3 つの組み込みアダプターは、キーが存在しない場合にエラーなしで空文字列を返します。有効期間はミリ秒単位で、正の最小値は 1ms です。ゼロは無期限、負の期間または `nil` は削除を意味します。`Remove` は削除数を返し、クラスター対応のためキーごとに削除します。複数キーの削除はアトミックではありません。クライアントの終了処理は呼び出し元が行います。go-zero は TTL 付きの書き込みに単一キーの `EVAL` を使用するため、その Redis 権限が必要です。

ストアを有効にすると、同じ発行者と主体に対する最新のログインまたは更新が古いトークンを置き換え、ログアウトでセッション記録を削除します。`WithStore` がない場合はステートレスであり、ログアウトでは失効できず、古いトークンも期限まで有効です。セッションの TTL は有効期間と更新期間の大きい方です。

## 実行可能な HTTP サンプル

リポジトリのルートから、まず `127.0.0.1:6379` で Redis を起動し、[サンプル](example/server.go)を実行します。

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

モードを切り替える前に実行中のサーバーを停止してください。クラスターモードには構成済みの Redis Cluster が必要で、独立した Redis サーバーを並べるだけでは動作しません。

| モード | コマンド |
| --- | --- |
| go-redis v8 · 単一ノード | `go run . -store redisv8` |
| go-redis v8 · クラスター | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · 単一ノード | `go run . -store redisv9` |
| go-redis v9 · クラスター | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · 単一ノード | `go run . -store gozero` |
| go-zero · クラスター | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · 単一ノード | `go run . -store goframe` |
| Redis なし／ステートレス | `go run . -store none` |

### 設定

| 引数／環境変数 | 既定値 | 説明 |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | HTTP 待ち受けアドレス |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | カンマ区切りの Redis 接続先アドレス |
| `-redis-db` | `0` | データベース番号。クラスターと go-zero は 0 のみ |
| `-cluster` | `false` | v8、v9、go-zero のクラスターモード |
| `JWT_SECRET` | 必須 | JWT 署名用シークレット |
| `REDIS_USERNAME` | 空 | Redis ACL ユーザー名。v8、v9、go-zero に対応 |
| `REDIS_PASSWORD` | 空 | Redis パスワード。すべての Redis モードに対応 |

GoFrame v1 のサンプルは `g.Redis("jwt")` と `adapter.NewRedis(...)` を使用し、単一ノード、パスワード認証、`-redis-db` に対応します。クラスターと ACL ユーザー名には対応しません。[ストア初期化](example/internal/demo/store.go)を参照してください。起動時に接続を確認し、Ctrl+C で終了処理を行います。

### HTTP 操作手順

`curl` と `jq` が必要です。サンプル専用の認証情報は `demo`／`demo-password` です。アプリケーションでは実際のユーザー認証に置き換えてください。

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

Redis を有効にした場合、ログアウト後の最後のリクエストは `401` を返します。更新後は新しいトークンを使用してください。`none` モードのログアウトは `501` を返し、トークンを失効させません。

| エンドポイント | 認証 | 用途 |
| --- | --- | --- |
| `GET /healthz` | — | HTTP の生存確認のみ。Redis の準備状態は確認しません |
| `POST /login` | — | サンプルの JSON 認証情報を検証してトークンを発行 |
| `GET /profile` | `Bearer` | トークンを検証してクレームを返す |
| `PUT /refresh` | `Bearer` | 有効なトークンを更新 |
| `DELETE /logout` | `Bearer` | 保存された現在のセッションを失効 |

[Postman コレクション](example/jwt.postman.json)をインポートしてください。コレクション変数 `base_url` と `token` を使用し、ログインと更新のトークンを自動保存します。Redis を使用するモードで実行してください。

## API の説明

詳細は [JWT 操作](jwt.go)、[設定オプション](options.go)、[HTTP ヘルパー](http.go)を参照してください。

| API | 用途 |
| --- | --- |
| `GenerateToken(subject, payload...)` | 文字列の主体と任意のカスタムクレームからトークンを発行 |
| `ParseToken(token)` / `RefreshToken(token)` | クレームの検証／置き換えトークンの発行 |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | ストアが設定されている場合にセッション記録を削除 |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | 単なる整数ではなく `time.Hour` などを指定。既定の有効期間／更新期間は 2h／1h で、通常の検証では両方の期限を確認 |
| `WithLookupLocations("header:Authorization")` | トークン取得元をカンマ区切りで指定：`header`、`query`、`cookie`、`form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | リクエストの検証／コンテキストに保存されたクレームの取得 |

`jwt.IsMissingToken`、`jwt.IsExpiredToken`、`jwt.IsInvalidToken`、`jwt.IsAuthElsewhere` または `errors.Is` でエラーを判定できます。サンプルは不正なログイン JSON に `400`、認証失敗に `401`、ストア障害に `503` を返します。

## テスト

```sh
go test ./...
cd example
go test ./...
```

miniredis により v8、v9、go-zero の単一ノード／クラスタークライアントと、GoFrame およびステートレス HTTP モードを検証します。有効期限、更新、失効、エラー処理を対象とします。Redis のシミュレーションテストであり、実際の複数ノード構成の検証を代替するものではありません。

## ライセンスと由来

[MIT](LICENSE)。[dobyte/jwt](https://github.com/dobyte/jwt) からのフォークで、元の著作権表示を保持しています。
