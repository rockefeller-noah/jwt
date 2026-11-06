# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

Frameworkunabhängige JWT-Bibliothek für Go mit HTTP-Hilfsfunktionen und Redis-gestützter Sitzungsprüfung. Unterstützt HS256/384/512, RS256/384/512 und ES256/384/512.

## Installation

Benötigt Go 1.25 oder neuer. Redis ist für die Bibliothek optional und für die Beispiele mit Redis-Speicherung erforderlich.

```sh
go get github.com/rockefeller-noah/jwt
```

## Schnellstart

Speichere dieses Programm als `main.go` in deiner Go-Anwendung:

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

Führe `JWT_SECRET=local-example-secret-change-me go run main.go` aus. Die erwartete Ausgabe ist `user-1`.

## Redis-Adapter

Übergib deinen vorhandenen Client an `jwt.WithStore(...)`. Importiere nur den benötigten Adapter:

| Client | Importpfad | Verwendung |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

Bei v8 und v9 akzeptiert `New` sowohl `*redis.Client` als auch `*redis.ClusterClient` über `redis.UniversalClient`, ebenso Sentinel-Clients. Es werden keine neuen Clients oder Verbindungspools erzeugt. Für go-zero wird ein vorhandener `*redis.Redis` übergeben, etwa `gozero.New(svcCtx.Redis)`. Die ursprüngliche GoFrame-Anbindung mit `adapter.NewRedis(g.Redis("jwt"))` bleibt unterstützt.

Die drei integrierten Adapter liefern bei fehlenden Schlüsseln eine leere Zeichenfolge ohne Fehler. Ablaufzeiten haben Millisekundenauflösung mit einem positiven Minimum von 1ms; null bedeutet unbegrenzt, eine negative Dauer oder ein `nil`-Wert löscht den Schlüssel. `Remove` liefert die Anzahl gelöschter Schlüssel und löscht sie für Cluster-Kompatibilität einzeln; mehrere Löschungen sind nicht atomar. Der aufrufende Code verwaltet das Schließen des Clients. go-zero verwendet Einzel-Schlüssel-`EVAL`, um Wert und TTL atomar zu schreiben; die entsprechende Redis-Berechtigung wird benötigt.

Mit Speicher ersetzt die neueste Anmeldung oder Erneuerung das vorherige Token desselben Ausstellers und Subjekts; die Abmeldung entfernt den Sitzungseintrag. Ohne `WithStore` sind Tokens zustandslos: Eine Abmeldung kann sie nicht widerrufen und alte Tokens bleiben bis zum Ablauf gültig. Die Sitzungs-TTL entspricht dem größeren Wert aus Gültigkeits- und Erneuerungsdauer.

## Ausführbares HTTP-Beispiel

Starte ausgehend vom Repository-Stammverzeichnis Redis auf `127.0.0.1:6379` und anschließend [das Beispiel](example/server.go):

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

Beende den laufenden Server vor dem Moduswechsel. Cluster-Modi benötigen einen eingerichteten Redis Cluster, nicht mehrere unabhängige Redis-Server.

| Modus | Befehl |
| --- | --- |
| go-redis v8 · Einzelknoten | `go run . -store redisv8` |
| go-redis v8 · Cluster | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · Einzelknoten | `go run . -store redisv9` |
| go-redis v9 · Cluster | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · Einzelknoten | `go run . -store gozero` |
| go-zero · Cluster | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · Einzelknoten | `go run . -store goframe` |
| Ohne Redis / zustandslos | `go run . -store none` |

### Konfiguration

| Parameter / Umgebungsvariable | Standardwert | Beschreibung |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | HTTP-Listen-Adresse |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | Kommagetrennte Redis-Einstiegsadressen |
| `-redis-db` | `0` | Datenbankindex; Cluster und go-zero benötigen 0 |
| `-cluster` | `false` | Cluster-Modus für v8, v9 und go-zero |
| `JWT_SECRET` | erforderlich | Geheimer JWT-Signaturschlüssel |
| `REDIS_USERNAME` | leer | Redis-ACL-Benutzername; v8, v9 und go-zero |
| `REDIS_PASSWORD` | leer | Redis-Passwort; alle Redis-Modi |

Das GoFrame-v1-Beispiel nutzt weiterhin `g.Redis("jwt")` mit `adapter.NewRedis(...)` und unterstützt Einzelknoten, Passwortauthentifizierung und `-redis-db`. Cluster und ACL-Benutzernamen werden nicht unterstützt. Siehe [Speicherinitialisierung](example/internal/demo/store.go). Das Beispiel prüft beim Start die Verbindung und behandelt das Beenden mit Ctrl+C.

### HTTP-Ablauf

Benötigt `curl` und `jq`. Das reine Beispielkonto lautet `demo` / `demo-password`; ersetze diese feste Prüfung in einer Anwendung durch deine Benutzerverwaltung.

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

Mit Redis liefert die letzte Anfrage nach der Abmeldung `401`. Verwende nach der Erneuerung immer das neue Token. Im Modus `none` liefert die Abmeldung `501` und widerruft keine Tokens.

| Endpunkt | Authentifizierung | Zweck |
| --- | --- | --- |
| `GET /healthz` | — | Prüft nur die HTTP-Erreichbarkeit, nicht die Redis-Bereitschaft |
| `POST /login` | — | JSON-Beispielzugangsdaten prüfen und ein Token ausstellen |
| `GET /profile` | `Bearer` | Token prüfen und seine Claims zurückgeben |
| `PUT /refresh` | `Bearer` | Ein noch gültiges Token erneuern |
| `DELETE /logout` | `Bearer` | Die aktuell gespeicherte Sitzung widerrufen |

Importiere [die Postman-Sammlung](example/jwt.postman.json). Sie verwendet die Sammlungsvariablen `base_url` und `token` und speichert Tokens nach Anmeldung und Erneuerung automatisch. Führe sie in einem Modus mit Redis-Speicher aus.

## API-Hinweise

Die vollständige API findest du unter [JWT-Operationen](jwt.go), [Optionen](options.go) und [HTTP-Hilfsfunktionen](http.go).

| API | Zweck |
| --- | --- |
| `GenerateToken(subject, payload...)` | Token für ein Textsubjekt und optionale eigene Claims ausstellen |
| `ParseToken(token)` / `RefreshToken(token)` | Claims prüfen / ein Ersatztoken ausstellen |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | Sitzungseintrag bei konfiguriertem Speicher entfernen |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | `time.Hour` statt Ganzzahlen ohne Einheit verwenden. Standard: 2h / 1h; die normale Prüfung berücksichtigt beide Zeitgrenzen |
| `WithLookupLocations("header:Authorization")` | Kommagetrennte Token-Quellen: `header`, `query`, `cookie`, `form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | Anfrage prüfen / im Anfragekontext gespeicherte Claims lesen |

Fehler lassen sich mit `jwt.IsMissingToken`, `jwt.IsExpiredToken`, `jwt.IsInvalidToken`, `jwt.IsAuthElsewhere` oder `errors.Is` unterscheiden. Das Beispiel liefert `400` bei fehlerhaftem Anmelde-JSON, `401` bei Authentifizierungsfehlern und `503` bei Speicherfehlern.

## Tests

```sh
go test ./...
cd example
go test ./...
```

Die miniredis-Tests prüfen Einzelknoten- und Cluster-Clients von v8, v9 und go-zero sowie die HTTP-Modi GoFrame und zustandslos. Sie decken Ablauf, Erneuerung, Widerruf und Fehler ab. Es handelt sich um simulierte Redis-Tests, nicht um Deployment-Tests gegen einen echten Mehrknotencluster.

## Lizenz und Herkunft

[MIT](LICENSE). Fork von [dobyte/jwt](https://github.com/dobyte/jwt); ursprüngliche Urheberrechtshinweise bleiben erhalten.
