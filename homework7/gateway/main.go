package main

import (
	"context"
	"log"
	"net/http"

	"gateway/internal/api"
	"ledger"
)

func main() {
	if err := ledger.Connect(context.Background()); err != nil {
		log.Fatalf("сервис Gateway не запущен: %v", err)
	}
	defer ledger.Close()

	mux := http.NewServeMux()
	api.Register(mux)

	addr := ":8080"
	log.Printf("сервис Gateway запущен на %s", addr)
	if err := http.ListenAndServe(addr, api.Logging(mux)); err != nil {
		log.Fatal(err)
	}
}
