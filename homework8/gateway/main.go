package main

import (
	"context"
	"log"
	"net/http"

	"gateway/internal/api"
	"ledger"
)

func main() {
	svc, closeFn, err := ledger.New(context.Background())
	if err != nil {
		log.Fatalf("сервис Gateway не запущен: %v", err)
	}
	defer closeFn()

	mux := http.NewServeMux()
	api.New(svc).Register(mux)

	addr := ":8080"
	log.Printf("сервис Gateway запущен на %s", addr)
	if err := http.ListenAndServe(addr, api.Logging(mux)); err != nil {
		log.Fatal(err)
	}
}
