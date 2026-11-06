package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example/internal/demo"
	"github.com/rockefeller-noah/jwt"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	var cfg demo.StoreConfig
	listen := flag.String("listen", "127.0.0.1:8888", "HTTP listen address")
	flag.StringVar(&cfg.Driver, "store", "redisv9", "redisv8, redisv9, gozero, goframe, or none")
	flag.StringVar(&cfg.Addrs, "redis-addrs", "127.0.0.1:6379", "comma-separated Redis host:port addresses")
	flag.IntVar(&cfg.DB, "redis-db", 0, "Redis database; clusters and go-zero require 0")
	flag.BoolVar(&cfg.Cluster, "cluster", false, "use Redis Cluster (v8, v9, or go-zero)")
	flag.Parse()
	cfg.Username = os.Getenv("REDIS_USERNAME")
	cfg.Password = os.Getenv("REDIS_PASSWORD")
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return fmt.Errorf("set JWT_SECRET before starting the example")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 5*time.Second)
	store, cleanup, err := demo.OpenStore(startup, cfg)
	cancel()
	if err != nil {
		return fmt.Errorf("initialize store: %w", err)
	}
	defer cleanup()
	auth, err := jwt.NewJWT(
		jwt.WithIssuer("jwt-example"),
		jwt.WithSecretKey(secret),
		jwt.WithValidDuration(time.Hour),
		jwt.WithRefreshDuration(time.Hour),
		jwt.WithLookupLocations("header:Authorization"),
		jwt.WithStore(store),
	)
	if err != nil {
		return fmt.Errorf("initialize JWT: %w", err)
	}
	server := &http.Server{
		Addr: *listen, Handler: demo.NewHandler(auth, store != nil),
		ReadHeaderTimeout: 5 * time.Second,
	}
	stopped := make(chan struct{})
	shutdownDone := make(chan struct{})
	defer close(stopped)
	go func() {
		defer close(shutdownDone)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdown); err != nil {
				log.Printf("HTTP shutdown: %v", err)
				_ = server.Close()
			}
		case <-stopped:
		}
	}()
	log.Printf("JWT example listening on http://%s (store=%s, cluster=%t)", *listen, cfg.Driver, cfg.Cluster)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	<-shutdownDone
	return nil
}
