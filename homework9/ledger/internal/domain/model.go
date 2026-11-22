// Пакет domain хранит сущности и контракты хранилища.
// Он не знает ни про PostgreSQL, ни про HTTP.
package domain

import (
	"context"
	"errors"
	"time"
)

// ErrBudgetExceeded возвращается, когда новая трата превышает лимит категории.
var ErrBudgetExceeded = errors.New("budget exceeded")

// ErrNotFound возвращается, когда запись в хранилище отсутствует.
var ErrNotFound = errors.New("не найдено")

// Transaction — расход.
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

// ReportRow — сумма расходов одной категории за интервал дат.
type ReportRow struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// BudgetRepository описывает операции с бюджетами, нужные бизнес-логике.
type BudgetRepository interface {
	Save(ctx context.Context, budget Budget) error
	FindByCategory(ctx context.Context, category string) (Budget, error)
	List(ctx context.Context) ([]Budget, error)
}

// ExpenseRepository описывает операции с расходами, нужные бизнес-логике.
type ExpenseRepository interface {
	Add(ctx context.Context, tx Transaction) (Transaction, error)
	List(ctx context.Context) ([]Transaction, error)
	SumByCategory(ctx context.Context, category string) (float64, error)
	Categories(ctx context.Context) ([]string, error)
	SumByCategoryPeriod(ctx context.Context, category string, from, to time.Time) (float64, error)
}

// Validatable — контракт проверки данных перед записью.
type Validatable interface {
	Validate() error
}

// CheckValid вызывает Validate у любого значения, которое реализует контракт.
func CheckValid(v Validatable) error {
	return v.Validate()
}
