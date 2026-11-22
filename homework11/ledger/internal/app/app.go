// Пакет app собирает сервис Ledger и не отдаёт наружу детали PostgreSQL и Redis.
package app

import (
	"context"
	"fmt"
	"log"

	"ledger/internal/cache"
	"ledger/internal/db"
	"ledger/internal/repository/pg"
	"ledger/internal/service"
)

// New открывает базу и Redis, собирает репозитории и возвращает готовый сервис.
// Миграции здесь не запускаются. При ошибке close равен nil.
func New(ctx context.Context) (service.Service, func(), error) {
	database, err := db.Open(ctx, db.DSN())
	if err != nil {
		return nil, nil, fmt.Errorf("подключение к PostgreSQL: %w", err)
	}

	cacheStore, err := cache.Open(ctx)
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("подключение к Redis: %w", err)
	}

	svc := service.New(
		pg.NewBudgetRepository(database),
		pg.NewExpenseRepository(database),
		cacheStore,
	)
	log.Println("подключение к PostgreSQL установлено")
	log.Println("подключение к Redis установлено")

	closeFn := func() {
		_ = cacheStore.Close()
		_ = database.Close()
	}
	return svc, closeFn, nil
}
