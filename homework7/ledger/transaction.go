package ledger

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrBudgetExceeded возвращается, когда новая трата превышает лимит категории.
var ErrBudgetExceeded = errors.New("budget exceeded")

// Transaction — расход, сохранённый в таблице expenses.
type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

// Validate проверяет поля транзакции. Дата имеет тип time.Time, отдельный разбор формата не нужен.
func (tx Transaction) Validate() error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}
	if tx.Category == "" {
		return errors.New("transaction category must not be empty")
	}
	return nil
}

// AddTransaction сохраняет расход. Категория без бюджета принимается без проверки лимита.
func AddTransaction(tx Transaction) (Transaction, error) {
	if err := tx.Validate(); err != nil {
		return Transaction{}, err
	}
	if err := requireDB(); err != nil {
		return Transaction{}, err
	}

	ctx := context.Background()
	var limit float64
	err := database.QueryRowContext(ctx, `
		SELECT limit_amount::float8
		FROM budgets
		WHERE category = $1
	`, tx.Category).Scan(&limit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, err
	}
	if err == nil {
		var spent float64
		if err := database.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0)::float8
			FROM expenses
			WHERE category = $1
		`, tx.Category).Scan(&spent); err != nil {
			return Transaction{}, err
		}
		if spent+tx.Amount > limit {
			return Transaction{}, ErrBudgetExceeded
		}
	}

	err = database.QueryRowContext(ctx, `
		INSERT INTO expenses (amount, category, description, date)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, tx.Amount, tx.Category, tx.Description, tx.Date).Scan(&tx.ID)
	if err != nil {
		return Transaction{}, err
	}
	return tx, nil
}

// ListTransactions возвращает расходы: сначала более поздние даты, затем больший id.
func ListTransactions() ([]Transaction, error) {
	if err := requireDB(); err != nil {
		return nil, err
	}

	rows, err := database.QueryContext(context.Background(), `
		SELECT id, amount::float8, category, COALESCE(description, ''), date
		FROM expenses
		ORDER BY date DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Transaction, 0)
	for rows.Next() {
		var item Transaction
		if err := rows.Scan(&item.ID, &item.Amount, &item.Category, &item.Description, &item.Date); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// Reset очищает таблицы и ключи кеша. Нужен тестам, чтобы сценарии не влияли друг на друга.
func Reset() {
	if database == nil {
		return
	}
	ctx := context.Background()
	_, _ = database.ExecContext(ctx, `DELETE FROM expenses`)
	_, _ = database.ExecContext(ctx, `DELETE FROM budgets`)
	if redisCache != nil {
		_ = redisCache.Del(ctx, budgetsCacheKey)
		_ = redisCache.DelByPrefix(ctx, "report:summary:")
	}
}
