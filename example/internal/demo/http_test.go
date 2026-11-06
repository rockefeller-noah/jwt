package demo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/rockefeller-noah/jwt"
)

func TestHTTPFlow(t *testing.T) {
	for _, cfg := range []StoreConfig{
		{Driver: "redisv8"}, {Driver: "redisv8", Cluster: true},
		{Driver: "redisv9"}, {Driver: "redisv9", Cluster: true},
		{Driver: "gozero"}, {Driver: "gozero", Cluster: true},
		{Driver: "goframe"}, {Driver: "none"},
	} {
		name := cfg.Driver
		if cfg.Cluster {
			name += "/cluster"
		}
		t.Run(name, func(t *testing.T) {
			s := miniredis.RunT(t)
			cfg.Addrs = s.Addr()
			store, cleanup, err := OpenStore(context.Background(), cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(cleanup)
			auth, err := jwt.NewJWT(jwt.WithSecretKey("test-secret"),
				jwt.WithValidDuration(time.Hour), jwt.WithStore(store))
			if err != nil {
				t.Fatal(err)
			}
			h := NewHandler(auth, store != nil)
			request(t, h, "GET", "/healthz", "", "", 200)
			request(t, h, "GET", "/profile", "", "", 401)
			request(t, h, "POST", "/login", "{", "", 400)
			request(t, h, "POST", "/login", `{"account":"demo","password":"bad"}`, "", 401)
			request(t, h, "POST", "/login", `{"account":"demo","password":"demo-password"} {}`, "", 400)
			token := request(t, h, "POST", "/login", `{"account":"demo","password":"demo-password"}`, "", 200)
			request(t, h, "GET", "/profile", "", token, 200)
			refreshed := request(t, h, "PUT", "/refresh", "", token, 200)
			request(t, h, "GET", "/profile", "", refreshed, 200)
			if store == nil {
				request(t, h, "DELETE", "/logout", "", refreshed, 501)
				request(t, h, "GET", "/profile", "", refreshed, 200)
				return
			}
			request(t, h, "GET", "/profile", "", token, 401)
			request(t, h, "DELETE", "/logout", "", refreshed, 200)
			request(t, h, "GET", "/profile", "", refreshed, 401)
			token = request(t, h, "POST", "/login", `{"account":"demo","password":"demo-password"}`, "", 200)
			s.SetError("ERR store unavailable")
			request(t, h, "GET", "/profile", "", token, 503)
		})
	}
}

func request(t *testing.T, h http.Handler, method, path, body, token string, want int) string {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("%s %s: status=%d, want %d, body=%s", method, path, w.Code, want, w.Body)
	}
	var response struct {
		Status int       `json:"status"`
		Data   jwt.Token `json:"data"`
	}
	decoder := json.NewDecoder(w.Body)
	if err := decoder.Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Status != want {
		t.Fatalf("JSON status=%d, want %d", response.Status, want)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("multiple response bodies: %v", err)
	}
	return response.Data.Token
}

func TestInvalidStoreConfig(t *testing.T) {
	for _, cfg := range []StoreConfig{
		{Driver: "unknown", Addrs: "localhost:6379"},
		{Driver: "redisv9", Addrs: "missing-port"},
		{Driver: "redisv8", Addrs: "localhost:6379,localhost:6380"},
		{Driver: "redisv9", Addrs: "localhost:6379", Cluster: true, DB: 1},
		{Driver: "gozero", Addrs: "localhost:6379", DB: 1},
		{Driver: "goframe", Addrs: "localhost:6379", Cluster: true},
	} {
		store, cleanup, err := OpenStore(context.Background(), cfg)
		cleanup()
		if err == nil || store != nil {
			t.Fatalf("config=%+v, store=%v, err=%v", cfg, store, err)
		}
	}
}
