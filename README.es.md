# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

Biblioteca JWT para Go independiente del framework, con utilidades HTTP y verificación de sesiones mediante Redis. Compatible con HS256/384/512, RS256/384/512 y ES256/384/512.

## Instalación

Requiere Go 1.25 o posterior. Redis es opcional para la biblioteca y necesario para los ejemplos que usan almacenamiento Redis.

```sh
go get github.com/rockefeller-noah/jwt
```

## Inicio rápido

Guarda este programa como `main.go` en tu aplicación Go:

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

Ejecuta `JWT_SECRET=local-example-secret-change-me go run main.go`. El resultado esperado es `user-1`.

## Adaptadores de Redis

Pasa tu cliente existente a `jwt.WithStore(...)`. Importa únicamente el adaptador necesario:

| Cliente | Ruta de importación | Uso |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

En v8 y v9, `New` acepta `*redis.Client` o `*redis.ClusterClient` mediante `redis.UniversalClient`, incluidos los clientes Sentinel. No crea clientes ni grupos de conexiones nuevos. Para go-zero, pasa un `*redis.Redis` existente, por ejemplo `gozero.New(svcCtx.Redis)`. La integración original de GoFrame con `adapter.NewRedis(g.Redis("jwt"))` sigue siendo compatible.

Los tres adaptadores integrados devuelven una cadena vacía sin error cuando la clave no existe. La caducidad tiene precisión de milisegundos, con un mínimo positivo de 1ms; cero indica persistencia, y una duración negativa o un valor `nil` elimina la clave. `Remove` devuelve el número de claves eliminadas y las borra individualmente para admitir clústeres; el borrado múltiple no es atómico. El cierre del cliente corresponde al código que lo creó. go-zero usa `EVAL` con una sola clave para escribir el valor y su TTL de forma atómica y necesita ese permiso de Redis.

Con almacenamiento, el último inicio de sesión o renovación sustituye al token anterior del mismo emisor y sujeto; cerrar sesión elimina su registro. Sin `WithStore`, los tokens no tienen estado: cerrar sesión no puede revocarlos y los tokens anteriores siguen siendo válidos hasta caducar. El TTL de la sesión es el mayor de los períodos de validez y renovación.

## Ejemplo HTTP ejecutable

Desde la raíz del repositorio, inicia Redis en `127.0.0.1:6379` y ejecuta [el ejemplo](example/server.go):

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

Detén el servidor antes de cambiar de modo. Los modos de clúster requieren un Redis Cluster configurado, no varios servidores Redis independientes.

| Modo | Comando |
| --- | --- |
| go-redis v8 · nodo único | `go run . -store redisv8` |
| go-redis v8 · clúster | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · nodo único | `go run . -store redisv9` |
| go-redis v9 · clúster | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · nodo único | `go run . -store gozero` |
| go-zero · clúster | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · nodo único | `go run . -store goframe` |
| Sin Redis / sin estado | `go run . -store none` |

### Configuración

| Parámetro / variable de entorno | Valor predeterminado | Descripción |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | Dirección de escucha HTTP |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | Direcciones iniciales de Redis separadas por comas |
| `-redis-db` | `0` | Índice de base de datos; clústeres y go-zero requieren 0 |
| `-cluster` | `false` | Modo de clúster para v8, v9 y go-zero |
| `JWT_SECRET` | obligatorio | Secreto para firmar JWT |
| `REDIS_USERNAME` | vacío | Usuario ACL de Redis; v8, v9 y go-zero |
| `REDIS_PASSWORD` | vacío | Contraseña de Redis; todos los modos Redis |

El ejemplo de GoFrame v1 conserva `g.Redis("jwt")` y `adapter.NewRedis(...)`; admite un nodo, autenticación por contraseña y `-redis-db`. No admite clústeres ni usuarios ACL. Consulta [la inicialización del almacenamiento](example/internal/demo/store.go). El ejemplo verifica la conexión al arrancar y gestiona el cierre con Ctrl+C.

### Flujo de llamadas HTTP

Requiere `curl` y `jq`. La cuenta exclusiva del ejemplo es `demo` / `demo-password`; sustituye esta comprobación fija por la autenticación real de tu aplicación.

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

Con Redis, la última petición devuelve `401` después de cerrar sesión. Usa siempre el token devuelto por la renovación. En modo `none`, cerrar sesión devuelve `501` y no revoca los tokens.

| Ruta | Autenticación | Función |
| --- | --- | --- |
| `GET /healthz` | — | Solo comprueba que HTTP responde, no que Redis esté disponible |
| `POST /login` | — | Validar las credenciales JSON del ejemplo y emitir un token |
| `GET /profile` | `Bearer` | Validar el token y devolver sus atributos |
| `PUT /refresh` | `Bearer` | Renovar un token todavía válido |
| `DELETE /logout` | `Bearer` | Revocar la sesión almacenada actual |

Importa [la colección de Postman](example/jwt.postman.json). Usa las variables de colección `base_url` y `token`, y guarda automáticamente los tokens de inicio de sesión y renovación. Ejecútala con un modo que use Redis.

## Notas de la API

Consulta [las operaciones JWT](jwt.go), [las opciones](options.go) y [las utilidades HTTP](http.go) para ver la API completa.

| API | Función |
| --- | --- |
| `GenerateToken(subject, payload...)` | Emitir un token para un sujeto de texto y atributos personalizados opcionales |
| `ParseToken(token)` / `RefreshToken(token)` | Validar atributos / emitir un token de sustitución |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | Eliminar el registro de sesión si hay almacenamiento configurado |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | Usa `time.Hour`, no enteros sin unidad. Valores predeterminados: 2h / 1h; la validación normal comprueba ambos límites |
| `WithLookupLocations("header:Authorization")` | Ubicaciones del token separadas por comas: `header`, `query`, `cookie`, `form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | Validar una petición / leer atributos guardados en su contexto |

Clasifica errores con `jwt.IsMissingToken`, `jwt.IsExpiredToken`, `jwt.IsInvalidToken`, `jwt.IsAuthElsewhere` o `errors.Is`. El ejemplo devuelve `400` para JSON de acceso mal formado, `401` para fallos de autenticación y `503` para fallos del almacenamiento.

## Pruebas

```sh
go test ./...
cd example
go test ./...
```

Las pruebas con miniredis cubren los clientes de nodo único y clúster de v8, v9 y go-zero, además de los modos HTTP GoFrame y sin estado. Comprueban caducidad, renovación, revocación y errores. Son pruebas simuladas de Redis, no pruebas de despliegue en un clúster real con varios nodos.

## Licencia y origen

[MIT](LICENSE). Derivado de [dobyte/jwt](https://github.com/dobyte/jwt); se conservan los avisos de derechos de autor originales.
