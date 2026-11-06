# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

Biblioteca JWT para Go independente de framework, com utilitários HTTP e verificação de sessões por Redis. Compatível com HS256/384/512, RS256/384/512 e ES256/384/512.

## Instalação

Requer Go 1.25 ou posterior. Redis é opcional para a biblioteca e necessário para os exemplos com armazenamento Redis.

```sh
go get github.com/rockefeller-noah/jwt
```

## Início rápido

Salve este programa como `main.go` na sua aplicação Go:

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

Execute `JWT_SECRET=local-example-secret-change-me go run main.go`. A saída esperada é `user-1`.

## Adaptadores Redis

Passe o cliente existente para `jwt.WithStore(...)`. Importe apenas o adaptador necessário:

| Cliente | Caminho de importação | Uso |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

Nas versões v8 e v9, `New` aceita `*redis.Client` ou `*redis.ClusterClient` por meio de `redis.UniversalClient`, incluindo clientes Sentinel. Nenhum cliente ou pool de conexões novo é criado. Para go-zero, passe um `*redis.Redis` existente, como `gozero.New(svcCtx.Redis)`. A integração original do GoFrame com `adapter.NewRedis(g.Redis("jwt"))` continua disponível.

Os três adaptadores integrados retornam uma string vazia sem erro quando a chave não existe. A expiração tem precisão de milissegundos, com mínimo positivo de 1ms; zero mantém o valor sem expiração, e uma duração negativa ou um valor `nil` exclui a chave. `Remove` retorna a quantidade de exclusões e remove as chaves individualmente para funcionar em cluster; múltiplas exclusões não são atômicas. Quem cria o cliente é responsável por fechá-lo. go-zero usa `EVAL` com uma única chave para gravar o valor e o TTL de forma atômica e precisa dessa permissão no Redis.

Com armazenamento, o login ou a renovação mais recente substitui o token anterior do mesmo emissor e sujeito; sair remove o registro da sessão. Sem `WithStore`, os tokens não mantêm estado: sair não pode revogá-los e os tokens anteriores continuam válidos até expirar. O TTL da sessão é o maior entre os períodos de validade e renovação.

## Exemplo HTTP executável

Na raiz do repositório, inicie Redis em `127.0.0.1:6379` e execute [o exemplo](example/server.go):

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

Pare o servidor antes de mudar de modo. Os modos de cluster exigem um Redis Cluster configurado, não vários servidores Redis independentes.

| Modo | Comando |
| --- | --- |
| go-redis v8 · nó único | `go run . -store redisv8` |
| go-redis v8 · cluster | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · nó único | `go run . -store redisv9` |
| go-redis v9 · cluster | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · nó único | `go run . -store gozero` |
| go-zero · cluster | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · nó único | `go run . -store goframe` |
| Sem Redis / sem estado | `go run . -store none` |

### Configuração

| Argumento / variável de ambiente | Padrão | Descrição |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | Endereço de escuta HTTP |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | Endereços iniciais do Redis separados por vírgulas |
| `-redis-db` | `0` | Índice do banco; clusters e go-zero exigem 0 |
| `-cluster` | `false` | Modo de cluster para v8, v9 e go-zero |
| `JWT_SECRET` | obrigatório | Segredo de assinatura JWT |
| `REDIS_USERNAME` | vazio | Usuário ACL do Redis; v8, v9 e go-zero |
| `REDIS_PASSWORD` | vazio | Senha do Redis; todos os modos Redis |

O exemplo GoFrame v1 mantém `g.Redis("jwt")` com `adapter.NewRedis(...)` e suporta nó único, autenticação por senha e `-redis-db`. Não suporta cluster nem usuários ACL. Consulte [a inicialização do armazenamento](example/internal/demo/store.go). O exemplo verifica a conexão na inicialização e trata o encerramento por Ctrl+C.

### Fluxo de chamadas HTTP

Requer `curl` e `jq`. A conta exclusiva do exemplo é `demo` / `demo-password`; substitua essa verificação fixa pela autenticação real da sua aplicação.

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

Com Redis, a última requisição retorna `401` após sair. Use sempre o token retornado pela renovação. No modo `none`, sair retorna `501` e não revoga tokens.

| Rota | Autenticação | Função |
| --- | --- | --- |
| `GET /healthz` | — | Verifica apenas se HTTP está ativo, não se Redis está pronto |
| `POST /login` | — | Validar as credenciais JSON do exemplo e emitir um token |
| `GET /profile` | `Bearer` | Validar o token e retornar seus atributos |
| `PUT /refresh` | `Bearer` | Renovar um token ainda válido |
| `DELETE /logout` | `Bearer` | Revogar a sessão armazenada atual |

Importe [a coleção Postman](example/jwt.postman.json). Ela usa as variáveis de coleção `base_url` e `token` e salva automaticamente os tokens de login e renovação. Execute-a com um modo que use Redis.

## Notas da API

Consulte [as operações JWT](jwt.go), [as opções](options.go) e [os utilitários HTTP](http.go) para a API completa.

| API | Função |
| --- | --- |
| `GenerateToken(subject, payload...)` | Emitir um token para um sujeito em texto e atributos personalizados opcionais |
| `ParseToken(token)` / `RefreshToken(token)` | Validar atributos / emitir um token substituto |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | Excluir o registro da sessão quando houver armazenamento configurado |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | Use `time.Hour`, não inteiros sem unidade. Padrões: 2h / 1h; a validação normal verifica os dois limites |
| `WithLookupLocations("header:Authorization")` | Locais do token separados por vírgulas: `header`, `query`, `cookie`, `form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | Validar uma requisição / ler atributos armazenados em seu contexto |

Classifique erros com `jwt.IsMissingToken`, `jwt.IsExpiredToken`, `jwt.IsInvalidToken`, `jwt.IsAuthElsewhere` ou `errors.Is`. O exemplo retorna `400` para JSON de login malformado, `401` para falha de autenticação e `503` para falha do armazenamento.

## Testes

```sh
go test ./...
cd example
go test ./...
```

Os testes com miniredis cobrem os clientes de nó único e cluster de v8, v9 e go-zero, além dos modos HTTP GoFrame e sem estado. Verificam expiração, renovação, revogação e erros. São testes simulados de Redis e não substituem a validação em um cluster real com vários nós.

## Licença e origem

[MIT](LICENSE). Derivado de [dobyte/jwt](https://github.com/dobyte/jwt); os avisos originais de direitos autorais foram mantidos.
