package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rockefeller-noah/jwt"
)

func TestJWTLifecycle(t *testing.T) {
	for _, name := range clientNames {
		t.Run(name, func(t *testing.T) {
			s, store := newTestStore(t, name)
			auth, err := jwt.NewJWT(
				jwt.WithIssuer("adapter-test"),
				jwt.WithSecretKey("test-secret"),
				jwt.WithValidDuration(time.Hour),
				jwt.WithRefreshDuration(2*time.Hour),
				jwt.WithStore(store),
			)
			if err != nil {
				t.Fatal(err)
			}
			token, err := auth.GenerateToken("user", jwt.Payload{"role": "admin"})
			if err != nil {
				t.Fatal(err)
			}
			if ttl := s.TTL("jwt:adapter-test:subject:user"); ttl != 2*time.Hour {
				t.Fatalf("subject TTL=%v, want 2h", ttl)
			}
			if claims, err := auth.ParseToken(token.Token); err != nil || claims.Subject() != "user" {
				t.Fatalf("ParseToken claims=%v, err=%v", claims, err)
			}
			refreshed, err := auth.RefreshToken(token.Token)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := auth.ParseToken(refreshed.Token); err != nil {
				t.Fatal(err)
			}
			if err := auth.DestroyToken(refreshed.Token); err != nil {
				t.Fatal(err)
			}
			if _, err := auth.ParseToken(refreshed.Token); !errors.Is(err, jwt.ErrInvalidToken) {
				t.Fatalf("revoked token: %v", err)
			}
			token, err = auth.GenerateToken("user")
			if err != nil {
				t.Fatal(err)
			}
			s.FastForward(2 * time.Hour)
			if _, err := auth.ParseToken(token.Token); !errors.Is(err, jwt.ErrInvalidToken) {
				t.Fatalf("expired store entry: %v", err)
			}
			// A different stored UUID must be reported as a login elsewhere.
			token, err = auth.GenerateToken("user")
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Set(context.Background(), "jwt:adapter-test:subject:user", "other-uuid", time.Hour); err != nil {
				t.Fatal(err)
			}
			if _, err := auth.ParseToken(token.Token); !errors.Is(err, jwt.ErrAuthElsewhere) {
				t.Fatalf("superseded token: %v", err)
			}
		})
	}
}
