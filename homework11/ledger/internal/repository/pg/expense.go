package pg

import (
	"context"
	"database/sql"
	"time"

	"ledger/internal/domain"
)

// ExpenseRepository хранит расходы в таблице expenses.
type ExpenseRepository struct {
	db *sql.DB
}

// NewExpenseRepository связывает репозиторий с пулом соединений.
func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

// Add вставляет расход и возвращает его с идентификатором из базы.
func (r *ExpenseRepository) Add(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO expenses (amount, category, description, date)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, tx.Amount, tx.Category, tx.Description, tx.Date).Scan(&tx.ID)
	if err != nil {
		return domain.Transaction{}, err
	}
	return tx, nil
}

// List возвращает расходы: сначала более поздние даты, затем больший id.
func (r *ExpenseRepository) List(ctx context.Context) ([]domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, amount::float8, category, COALESCE(description, ''), date
		FROM expenses
		ORDER BY date DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.Transaction, 0)
	for rows.Next() {
		var item domain.Transaction
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

// SumByCategory считает уже сохранённую сумму расходов категории.
func (r *ExpenseRepository) SumByCategory(ctx context.Context, category string) (float64, error) {
	var spent float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)::float8
		FROM expenses
		WHERE category = $1
	`, category).Scan(&spent)
	return spent, err
}

// Categories возвращает категории расходов и бюджетов без повторов.
func (r *ExpenseRepository) Categories(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category FROM expenses
		UNION
		SELECT category FROM budgets
		ORDER BY category
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		result = append(result, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// SumByCategoryPeriod считает сумму расходов категории за включительный интервал дат.
func (r *ExpenseRepository) SumByCategoryPeriod(ctx context.Context, category string, from, to time.Time) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)::float8
		FROM expenses
		WHERE category = $1 AND date >= $2 AND date <= $3
	`, category, from, to).Scan(&total)
	return total, err
}
