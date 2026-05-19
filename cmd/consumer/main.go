package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Alpin-A/prism/pkg/db"
	"github.com/Alpin-A/prism/pkg/metrics"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	consumer, err := metrics.NewConsumer(
		getenv("KAFKA_BROKER", "localhost:9092"),
		getenv("KAFKA_GROUP_ID", "prism-metric-consumer"),
		pool,
	)
	if err != nil {
		log.Fatalf("creating consumer: %v", err)
	}

	// Cancel the context on SIGINT or SIGTERM so the consumer shuts down cleanly.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("shutting down consumer...")
		cancel()
	}()

	if err := consumer.Run(ctx); err != nil {
		log.Fatalf("consumer error: %v", err)
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
