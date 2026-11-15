package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ledger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := ledger.Connect(ctx); err != nil {
		log.Fatalf("сервис Ledger не запущен: %v", err)
	}
	defer ledger.Close()

	log.Println("сервис Ledger запущен")
	<-ctx.Done()
	log.Println("сервис Ledger остановлен")
}
