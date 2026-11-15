package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"ledger/internal/cache"
	"ledger/internal/db"
)

// Соединения живут на уровне пакета: Gateway и процесс Ledger пользуются одним набором функций.
var (
	database   *sql.DB
	redisCache *cache.Store
)

// Connect открывает PostgreSQL и Redis. При ошибке уже открытые ресурсы закрываются.
func Connect(ctx context.Context) error {
	databaseConn, err := db.Open(ctx, db.DSN())
	if err != nil {
		return fmt.Errorf("подключение к PostgreSQL: %w", err)
	}

	cacheStore, err := cache.Open(ctx)
	if err != nil {
		_ = databaseConn.Close()
		return fmt.Errorf("подключение к Redis: %w", err)
	}

	database = databaseConn
	redisCache = cacheStore
	log.Println("подключение к PostgreSQL установлено")
	log.Println("подключение к Redis установлено")
	return nil
}

// Ready сообщает, установлено ли соединение с базой.
func Ready() bool {
	return database != nil
}

// Close освобождает соединения.
func Close() {
	if redisCache != nil {
		_ = redisCache.Close()
		redisCache = nil
	}
	if database != nil {
		_ = database.Close()
		database = nil
	}
}

func requireDB() error {
	if database == nil {
		return fmt.Errorf("база данных не подключена")
	}
	return nil
}
