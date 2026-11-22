// Пакет pg реализует доменные репозитории поверх PostgreSQL.
package pg

import (
	"context"
	"database/sql"
	"errors"

	"ledger/internal/domain"
)

// BudgetRepository хранит бюджеты в таблице budgets.
type BudgetRepository struct {
	db *sql.DB
}

// NewBudgetRepository связывает репозиторий с пулом соединений.
func NewBudgetRepository(db *sql.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

// Save добавляет бюджет или обновляет лимит существующей категории.
func (r *BudgetRepository) Save(ctx context.Context, budget domain.Budget) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO budgets (category, limit_amount)
		VALUES ($1, $2)
		ON CONFLICT (category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount
	`, budget.Category, budget.Limit)
	return err
}

// FindByCategory возвращает бюджет категории или domain.ErrNotFound.
func (r *BudgetRepository) FindByCategory(ctx context.Context, category string) (domain.Budget, error) {
	var budget domain.Budget
	err := r.db.QueryRowContext(ctx, `
		SELECT category, limit_amount::float8
		FROM budgets
		WHERE category = $1
	`, category).Scan(&budget.Category, &budget.Limit)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Budget{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Budget{}, err
	}
	return budget, nil
}

// List возвращает все бюджеты по алфавиту категорий.
func (r *BudgetRepository) List(ctx context.Context) ([]domain.Budget, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category, limit_amount::float8
		FROM budgets
		ORDER BY category
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.Budget, 0)
	for rows.Next() {
		var item domain.Budget
		if err := rows.Scan(&item.Category, &item.Limit); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
