package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Alpin-A/prism/pkg/api"
	"github.com/Alpin-A/prism/pkg/db"
	"github.com/Alpin-A/prism/pkg/experiment"
	"github.com/Alpin-A/prism/pkg/flags"
	"github.com/Alpin-A/prism/pkg/metrics"
	"github.com/Alpin-A/prism/pkg/statsclient"
)

func main() {
	ctx := context.Background()

	dbPort, err := strconv.Atoi(getenv("DB_PORT", "5432"))
	if err != nil {
		log.Fatalf("invalid DB_PORT: %v", err)
	}

	pool, err := db.NewPool(ctx, db.Config{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     dbPort,
		User:     requireenv("DB_USER"),
		Password: requireenv("DB_PASSWORD"),
		DBName:   requireenv("DB_NAME"),
		SSLMode:  getenv("DB_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	defer pool.Close()

	store := experiment.NewStore(pool)

	publisher, err := metrics.NewPublisher(getenv("KAFKA_BROKER", "localhost:9092"))
	if err != nil {
		log.Fatalf("creating kafka publisher: %v", err)
	}
	defer publisher.Close()

	statsAddr := getenv("STATS_GRPC_ADDR", "localhost:50051")
	sc, err := statsclient.New(statsAddr)
	if err != nil {
		log.Fatalf("connecting to stats service: %v", err)
	}
	defer sc.Close()

	flagStore := flags.NewStore(pool)
	flagEvaluator := flags.NewEvaluator(flagStore)

	router := api.NewRouter(store, publisher, sc, flagStore, flagEvaluator)

	addr := getenv("ADDR", ":8080")
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		log.Printf("prism-api listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
}

func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func requireenv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Fatalf("required environment variable %q is not set", key)
	return ""
}
