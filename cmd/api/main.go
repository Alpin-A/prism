package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Alpin-A/prism/pkg/api"
	"github.com/Alpin-A/prism/pkg/db"
	"github.com/Alpin-A/prism/pkg/experiment"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPool(ctx, db.Config{
		Host:     getenv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getenv("DB_USER", ""),
		Password: requireenv("DB_PASSWORD"),
		DBName:   getenv("DB_NAME", ""),
	})
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	defer pool.Close()

	store := experiment.NewStore(pool)
	router := api.NewRouter(store)

	addr := getenv("ADDR", ":8080")
	log.Printf("prism-api listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
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
