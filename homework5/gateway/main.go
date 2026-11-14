package main

import (
	"log"
	"net/http"

	"gateway/internal/api"
)

func main() {
	mux := http.NewServeMux()
	api.Register(mux)

	addr := ":8080"
	log.Printf("Gateway service started on %s", addr)
	log.Fatal(http.ListenAndServe(addr, api.Logging(mux)))
}
