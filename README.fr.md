# JWT

[English](README.md) · [简体中文](README.zh-CN.md) · [繁體中文](README.zh-TW.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md) · [Français](README.fr.md) · [Português](README.pt.md) · [Deutsch](README.de.md)

Bibliothèque JWT pour Go indépendante des frameworks, avec des utilitaires HTTP et une vérification des sessions via Redis. Prend en charge HS256/384/512, RS256/384/512 et ES256/384/512.

## Installation

Nécessite Go 1.25 ou une version ultérieure. Redis est facultatif pour la bibliothèque et requis pour les exemples utilisant le stockage Redis.

```sh
go get github.com/rockefeller-noah/jwt
```

## Démarrage rapide

Enregistrez ce programme dans `main.go` au sein de votre application Go :

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

Exécutez `JWT_SECRET=local-example-secret-change-me go run main.go`. Le résultat attendu est `user-1`.

## Adaptateurs Redis

Transmettez votre client existant à `jwt.WithStore(...)`. Importez uniquement l’adaptateur nécessaire :

| Client | Chemin d’importation | Utilisation |
| --- | --- | --- |
| go-redis v8 | `github.com/rockefeller-noah/jwt/adapter/redisv8` | `jwt.WithStore(redisv8.New(client))` |
| go-redis v9 | `github.com/rockefeller-noah/jwt/adapter/redisv9` | `jwt.WithStore(redisv9.New(client))` |
| go-zero | `github.com/rockefeller-noah/jwt/adapter/gozero` | `jwt.WithStore(gozero.New(client))` |
| GoFrame v1 | `github.com/gogf/gcache-adapter/adapter` | `jwt.WithStore(adapter.NewRedis(g.Redis("jwt")))` |

Pour v8 et v9, `New` accepte `*redis.Client` ou `*redis.ClusterClient` via `redis.UniversalClient`, ainsi que les clients Sentinel. Aucun nouveau client ni pool de connexions n’est créé. Pour go-zero, transmettez un `*redis.Redis` existant, par exemple `gozero.New(svcCtx.Redis)`. L’intégration GoFrame d’origine avec `adapter.NewRedis(g.Redis("jwt"))` reste prise en charge.

Les trois adaptateurs intégrés renvoient une chaîne vide sans erreur pour une clé absente. L’expiration est précise à la milliseconde, avec un minimum positif de 1ms ; zéro conserve la valeur sans expiration, une durée négative ou une valeur `nil` supprime la clé. `Remove` renvoie le nombre de suppressions et traite chaque clé séparément pour fonctionner en cluster ; les suppressions multiples ne sont pas atomiques. Le code appelant gère la fermeture du client. go-zero utilise `EVAL` sur une seule clé pour écrire atomiquement la valeur et son TTL ; cette permission Redis est requise.

Avec un stockage, la dernière connexion ou le dernier renouvellement remplace l’ancien jeton pour le même émetteur et le même sujet ; la déconnexion supprime la session. Sans `WithStore`, les jetons sont sans état : la déconnexion ne peut pas les révoquer et les anciens jetons restent valides jusqu’à expiration. Le TTL de session est le plus grand des délais de validité et de renouvellement.

## Exemple HTTP exécutable

Depuis la racine du dépôt, démarrez Redis sur `127.0.0.1:6379`, puis lancez [l’exemple](example/server.go) :

```sh
cd example
export JWT_SECRET='local-example-secret-change-me'
go run . -store redisv9
```

Arrêtez le serveur avant de changer de mode. Les modes cluster nécessitent un Redis Cluster configuré, et non plusieurs serveurs Redis indépendants.

| Mode | Commande |
| --- | --- |
| go-redis v8 · nœud unique | `go run . -store redisv8` |
| go-redis v8 · cluster | `go run . -store redisv8 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-redis v9 · nœud unique | `go run . -store redisv9` |
| go-redis v9 · cluster | `go run . -store redisv9 -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| go-zero · nœud unique | `go run . -store gozero` |
| go-zero · cluster | `go run . -store gozero -cluster -redis-addrs 127.0.0.1:7000,127.0.0.1:7001` |
| GoFrame v1 · nœud unique | `go run . -store goframe` |
| Sans Redis / sans état | `go run . -store none` |

### Configuration

| Argument / variable d’environnement | Valeur par défaut | Description |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8888` | Adresse d’écoute HTTP |
| `-store` | `redisv9` | `redisv8`, `redisv9`, `gozero`, `goframe`, `none` |
| `-redis-addrs` | `127.0.0.1:6379` | Adresses initiales Redis séparées par des virgules |
| `-redis-db` | `0` | Numéro de base ; les clusters et go-zero exigent 0 |
| `-cluster` | `false` | Mode cluster pour v8, v9 et go-zero |
| `JWT_SECRET` | obligatoire | Secret de signature JWT |
| `REDIS_USERNAME` | vide | Utilisateur ACL Redis ; v8, v9 et go-zero |
| `REDIS_PASSWORD` | vide | Mot de passe Redis ; tous les modes Redis |

L’exemple GoFrame v1 conserve `g.Redis("jwt")` avec `adapter.NewRedis(...)` et prend en charge un nœud unique, l’authentification par mot de passe et `-redis-db`. Les clusters et les utilisateurs ACL ne sont pas pris en charge. Voir [l’initialisation du stockage](example/internal/demo/store.go). L’exemple vérifie la connexion au démarrage et gère l’arrêt par Ctrl+C.

### Parcours HTTP

Nécessite `curl` et `jq`. Le compte de démonstration est `demo` / `demo-password` ; remplacez cette vérification fixe par l’authentification réelle de votre application.

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

Avec Redis, la dernière requête renvoie `401` après la déconnexion. Utilisez toujours le jeton renvoyé par le renouvellement. En mode `none`, la déconnexion renvoie `501` et ne révoque aucun jeton.

| Route | Authentification | Fonction |
| --- | --- | --- |
| `GET /healthz` | — | Vérifie uniquement la disponibilité HTTP, pas celle de Redis |
| `POST /login` | — | Vérifier les identifiants JSON de démonstration et émettre un jeton |
| `GET /profile` | `Bearer` | Vérifier le jeton et renvoyer ses attributs |
| `PUT /refresh` | `Bearer` | Renouveler un jeton encore valide |
| `DELETE /logout` | `Bearer` | Révoquer la session stockée actuelle |

Importez [la collection Postman](example/jwt.postman.json). Elle utilise les variables de collection `base_url` et `token` et enregistre automatiquement les jetons de connexion et de renouvellement. Exécutez-la avec un mode utilisant Redis.

## Notes sur l’API

Consultez [les opérations JWT](jwt.go), [les options](options.go) et [les utilitaires HTTP](http.go) pour l’API complète.

| API | Fonction |
| --- | --- |
| `GenerateToken(subject, payload...)` | Émettre un jeton pour un sujet textuel et des attributs personnalisés facultatifs |
| `ParseToken(token)` / `RefreshToken(token)` | Vérifier les attributs / émettre un jeton de remplacement |
| `DestroyToken(token)` / `DestroyTokenBySubject(subject)` | Supprimer la session lorsqu’un stockage est configuré |
| `WithValidDuration(duration)` / `WithRefreshDuration(duration)` | Utiliser `time.Hour`, pas d’entiers sans unité. Valeurs par défaut : 2h / 1h ; la vérification normale contrôle les deux limites |
| `WithLookupLocations("header:Authorization")` | Emplacements du jeton séparés par des virgules : `header`, `query`, `cookie`, `form` |
| `auth.Http().Middleware(r)` / `auth.Http().ParseToken(r)` | Vérifier une requête / lire les attributs mis en cache dans son contexte |

Classez les erreurs avec `jwt.IsMissingToken`, `jwt.IsExpiredToken`, `jwt.IsInvalidToken`, `jwt.IsAuthElsewhere` ou `errors.Is`. L’exemple renvoie `400` pour un JSON de connexion mal formé, `401` pour un échec d’authentification et `503` pour une panne du stockage.

## Tests

```sh
go test ./...
cd example
go test ./...
```

Les tests miniredis couvrent les clients à nœud unique et en cluster de v8, v9 et go-zero, ainsi que les modes HTTP GoFrame et sans état. Ils vérifient l’expiration, le renouvellement, la révocation et les erreurs. Ces tests simulent Redis et ne remplacent pas une validation sur un véritable cluster multinœud.

## Licence et origine

[MIT](LICENSE). Dérivé de [dobyte/jwt](https://github.com/dobyte/jwt) ; les mentions de droits d’auteur d’origine sont conservées.
