# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

Go용 프레임워크 독립적인 JWT 라이브러리입니다. HTTP 도우미와 Redis 기반 세션 검증을 제공하며 HS256/384/512, RS256/384/512, ES256/384/512를 지원합니다.

## 설치

Go 1.25 이상이 필요합니다. 라이브러리 자체에서는 Redis가 선택 사항이지만 Redis 저장소 예제에는 접속 가능한 Redis 서버가 필요합니다.

```sh
go get github.com/rockefeller-noah/jwt
```

## 빠른 시작

Go 애플리케이션에서 다음 프로그램을 `main.go`로 저장하세요.

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

`JWT_SECRET=local-example-secret-change-me go run main.go`를 실행하면 `user-1`이 출력됩니다.

## Redis 어댑터

`jwt.WithStore(...)`에 기존 클라이언트를 전달하세요. 필요한 어댑터만 가져오면 됩니다.

| 클라이언트 | 가져오기 경로 | 사용 방법 |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

v8과 v9의 `New`는 `redis.UniversalClient`를 통해 `*redis.Client`와 `*redis.ClusterClient`를 모두 받으며 Sentinel 클라이언트도 지원합니다. 새 클라이언트나 연결 풀을 만들지 않습니다. go-zero에서는 `gozero.New(svcCtx.Redis)`처럼 기존 `*redis.Redis`를 전달하세요. 기존 GoFrame의 `adapter.NewRedis(g.Redis("jwt"))` 연동도 계속 지원합니다.

세 내장 어댑터는 키가 없으면 오류 없이 빈 문자열을 반환합니다. 만료 시간은 밀리초 단위이며 양수의 최솟값은 1ms입니다. 0은 무기한 저장, 음수 기간 또는 `nil` 값은 삭제를 의미합니다. `Remove`는 삭제 수를 반환하고 클러스터 호환성을 위해 키를 개별 삭제하므로 여러 키의 삭제는 원자적이지 않습니다. 클라이언트 종료는 호출자가 담당합니다. go-zero는 TTL과 값을 원자적으로 기록하기 위해 단일 키 `EVAL`을 사용하므로 해당 Redis 권한이 필요합니다.

저장소를 사용하면 같은 발급자와 주체의 최신 로그인 또는 갱신이 이전 토큰을 대체하며 로그아웃 시 세션 기록을 삭제합니다. `WithStore`가 없으면 무상태 토큰이므로 로그아웃으로 취소할 수 없고 이전 토큰도 만료 전까지 유효합니다. 세션 TTL은 유효 기간과 갱신 기간 중 큰 값입니다.

## 실행 가능한 HTTP 예제

저장소 루트에서 시작합니다. 먼저 `127.0.0.1:6379`에 Redis를 실행한 뒤 [예제](example/server.go)를 실행하세요.

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

모드를 바꾸기 전에 실행 중인 서버를 종료하세요. 클러스터 모드에는 구성된 Redis Cluster가 필요하며 독립 Redis 서버 여러 개로 대체할 수 없습니다.

| 모드 | 명령 |
| --- | --- |
| go-redis v8 · 단일 노드 | `go run . -store redisv8` |
| go-redis v8 · 클러스터 | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · 단일 노드 | `go run . -store redisv9` |
| go-redis v9 · 클러스터 | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · 단일 노드 | `go run . -store gozero` |
| go-zero · 클러스터 | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · 단일 노드 | `go run . -store goframe` |
| Redis 없음／무상태 | `go run . -store none` |

### 설정

| 인자／환경 변수 | 기본값 | 설명 |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | HTTP 수신 주소 |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | 쉼표로 구분한 Redis 접속 주소 |
| `-redis-db` | `0` | 데이터베이스 번호. 클러스터와 go-zero는 0만 허용 |
| `-cluster` | `false` | v8, v9, go-zero의 클러스터 모드 |
| `JWT_SECRET` | 필수 | JWT 서명 비밀 키 |
| `REDIS_USERNAME` | 빈 값 | Redis ACL 사용자 이름. v8, v9, go-zero 지원 |
| `REDIS_PASSWORD` | 빈 값 | Redis 암호. 모든 Redis 모드 지원 |

GoFrame v1 예제는 `g.Redis("jwt")`와 `adapter.NewRedis(...)`를 사용하며 단일 노드 Redis, 암호 인증, `-redis-db`를 지원합니다. 클러스터와 ACL 사용자 이름은 지원하지 않습니다. [저장소 초기화](example/internal/demo/store.go)를 참고하세요. 시작 시 연결을 확인하고 Ctrl+C로 종료 처리합니다.

### HTTP 호출 흐름

`curl`과 `jq`가 필요합니다. 예제 전용 계정은 `demo`／`demo-password`이며 애플리케이션에서는 실제 사용자 인증으로 교체하세요.

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

Redis 사용 시 로그아웃 후 마지막 요청은 `401`을 반환합니다. 갱신 후에는 새 토큰을 사용하세요. `none` 모드의 로그아웃은 `501`을 반환하며 토큰을 취소하지 않습니다.

| 엔드포인트 | 인증 | 용도 |
| --- | --- | --- |
| `GET /healthz` | — | HTTP 생존 여부만 확인하며 Redis 준비 상태는 확인하지 않음 |
| `POST /login` | — | 예제 JSON 자격 증명을 검증하고 토큰 발급 |
| `GET /profile` | `Bearer` | 토큰을 검증하고 클레임 반환 |
| `PUT /refresh` | `Bearer` | 아직 유효한 토큰 갱신 |
| `DELETE /logout` | `Bearer` | 현재 저장된 세션 취소 |

[Postman 컬렉션](example/jwt.postman.json)을 가져오세요. 컬렉션 변수 `base_url`과 `token`을 사용하며 로그인과 갱신 토큰을 자동 저장합니다. Redis 저장소 모드에서 실행하세요.

## API 설명

전체 API는 [JWT 작업](jwt.go), [설정 옵션](options.go), [HTTP 도우미](http.go)를 참고하세요.

| API | 용도 |
| --- | --- |
| `GenerateToken(subject, payload...)` | 문자열 주체와 선택적 사용자 정의 클레임으로 토큰 발급 |
| `ParseToken(token)` / `RefreshToken(token)` | 클레임 검증／대체 토큰 발급 |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | 저장소가 설정된 경우 세션 기록 삭제 |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | 단순 정수가 아닌 `time.Hour` 같은 단위 사용. 기본 유효 기간／갱신 기간은 2h／1h이며 일반 검증은 두 제한을 모두 확인 |
| `WithLookupLocations("header:Authorization")` | 쉼표로 구분한 토큰 조회 위치: `header`, `query`, `cookie`, `form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | 요청 검증／요청 컨텍스트에 저장된 클레임 읽기 |

`jwt.IsMissingToken`, `jwt.IsExpiredToken`, `jwt.IsInvalidToken`, `jwt.IsAuthElsewhere` 또는 `errors.Is`로 오류를 구분할 수 있습니다. 예제는 잘못된 로그인 JSON에 `400`, 인증 실패에 `401`, 저장소 오류에 `503`을 반환합니다.

## 테스트

```sh
go test ./...
cd example
go test ./...
```

miniredis로 v8, v9, go-zero의 단일 노드／클러스터 클라이언트와 GoFrame 및 무상태 HTTP 모드를 검사합니다. 만료, 갱신, 취소, 오류 처리를 포함합니다. Redis 모의 테스트이며 실제 다중 노드 클러스터 배포 검증을 대체하지 않습니다.

## 라이선스와 출처

[MIT](LICENSE). [dobyte/jwt](https://github.com/dobyte/jwt)에서 포크했으며 원저작자의 저작권 고지를 유지합니다.
