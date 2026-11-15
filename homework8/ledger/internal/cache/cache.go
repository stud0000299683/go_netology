// Пакет cache хранит короткие JSON-ответы в Redis.
package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store — клиент Redis с короткими операциями get/set/del.
type Store struct {
	client *redis.Client
}

// Open читает REDIS_ADDR, REDIS_DB и REDIS_PASSWORD и проверяет соединение.
func Open(ctx context.Context) (*Store, error) {
	db, err := strconv.Atoi(envOr("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("REDIS_DB: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     envOr("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("проверка соединения: %w", err)
	}
	return &Store{client: client}, nil
}

// Get возвращает строку. Второй результат false означает, что ключа нет.
func (s *Store) Get(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.client == nil {
		return "", false, nil
	}
	value, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// Set сохраняет значение на заданное время.
func (s *Store) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Set(ctx, key, value, ttl).Err()
}

// Del удаляет ключи. Отсутствие ключа не считается ошибкой.
func (s *Store) Del(ctx context.Context, keys ...string) error {
	if s == nil || s.client == nil || len(keys) == 0 {
		return nil
	}
	return s.client.Del(ctx, keys...).Err()
}

// DelByPrefix удаляет ключи по шаблону, чтобы сбросить кеш отчётов в тестах.
func (s *Store) DelByPrefix(ctx context.Context, prefix string) error {
	if s == nil || s.client == nil {
		return nil
	}
	var keys []string
	iter := s.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return s.Del(ctx, keys...)
}

// Close закрывает клиент.
func (s *Store) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
