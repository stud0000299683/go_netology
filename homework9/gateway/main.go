package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"gateway/internal/api"
	"ledger"
)

func main() {
	svc, closeFn, err := ledger.New(context.Background())
	if err != nil {
		log.Fatalf("сервис Gateway не запущен: %v", err)
	}
	defer closeFn()

	timeout := api.DefaultRequestTimeout
	if value := os.Getenv("REQUEST_TIMEOUT"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			log.Fatalf("REQUEST_TIMEOUT=%q: нужна положительная длительность, например 2s", value)
		}
		timeout = parsed
	}

	mux := http.NewServeMux()
	api.New(svc).Register(mux)

	addr := ":8080"
	log.Printf("сервис Gateway запущен на %s, таймаут запроса %s", addr, timeout)
	handler := api.Logging(api.WithTimeout(timeout)(mux))
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
