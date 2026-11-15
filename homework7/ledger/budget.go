package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"
)

const (
	budgetsCacheKey = "budgets:all"
	budgetsCacheTTL = 30 * time.Second
)

// Budget — лимит трат на одну категорию.
// Period оставлен для совместимости с прежним API и в базе не хранится.
type Budget struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period,omitempty"`
}

// Validate проверяет, что бюджет можно сохранить.
func (b Budget) Validate() error {
	if b.Limit <= 0 {
		return errors.New("budget limit must be positive")
	}
	if b.Category == "" {
		return errors.New("budget category must not be empty")
	}
	return nil
}

// SetBudget создаёт или обновляет лимит категории.
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := requireDB(); err != nil {
		return err
	}

	_, err := database.ExecContext(context.Background(), `
		INSERT INTO budgets (category, limit_amount)
		VALUES ($1, $2)
		ON CONFLICT (category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount
	`, b.Category, b.Limit)
	if err != nil {
		return err
	}

	// Список бюджетов после записи сразу устаревает, поэтому ключ удаляется, а не ждёт TTL.
	if err := redisCache.Del(context.Background(), budgetsCacheKey); err != nil {
		log.Printf("не удалось сбросить кеш бюджетов: %v", err)
	}
	return nil
}

// ListBudgets возвращает бюджеты по алфавиту. Повторный вызов в течение TTL читает Redis.
func ListBudgets() ([]Budget, error) {
	if err := requireDB(); err != nil {
		return nil, err
	}
	ctx := context.Background()

	if raw, ok, err := redisCache.Get(ctx, budgetsCacheKey); err != nil {
		log.Printf("кеш бюджетов недоступен, читаю базу: %v", err)
	} else if ok {
		var cached []Budget
		if err := json.Unmarshal([]byte(raw), &cached); err != nil {
			log.Printf("кеш бюджетов повреждён, читаю базу: %v", err)
		} else {
			log.Printf("кеш бюджетов: попадание, ключ %s", budgetsCacheKey)
			return cached, nil
		}
	}

	rows, err := database.QueryContext(ctx, `
		SELECT category, limit_amount::float8
		FROM budgets
		ORDER BY category
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Budget, 0)
	for rows.Next() {
		var item Budget
		if err := rows.Scan(&item.Category, &item.Limit); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if err := redisCache.Set(ctx, budgetsCacheKey, string(payload), budgetsCacheTTL); err != nil {
		log.Printf("не удалось записать кеш бюджетов: %v", err)
	} else {
		log.Printf("кеш бюджетов: промах, ключ %s", budgetsCacheKey)
	}
	return result, nil
}
