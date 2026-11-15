// Пакет service применяет правила учёта и обращается к хранилищу только через интерфейсы домена.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"ledger/internal/domain"
)

const (
	budgetsCacheKey = "budgets:all"
	budgetsCacheTTL = 30 * time.Second
	reportCacheTTL  = 30 * time.Second
)

// Service — операции, которые нужны HTTP-слою.
type Service interface {
	SetBudget(ctx context.Context, budget domain.Budget) error
	ListBudgets(ctx context.Context) ([]domain.Budget, error)
	AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error)
	ListTransactions(ctx context.Context) ([]domain.Transaction, error)
	GetReportSummary(ctx context.Context, from, to time.Time) ([]domain.ReportRow, error)
}

// Cache хранит короткие JSON-ответы. Конкретный Redis сервис не импортирует.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// Ledger связывает правила бюджета с репозиториями.
type Ledger struct {
	budgets  domain.BudgetRepository
	expenses domain.ExpenseRepository
	cache    Cache
}

// New собирает сервис из переданных хранилищ. cache может быть nil.
func New(budgets domain.BudgetRepository, expenses domain.ExpenseRepository, cache Cache) Service {
	return &Ledger{budgets: budgets, expenses: expenses, cache: cache}
}

// SetBudget проверяет бюджет и сохраняет его. Кеш списка после записи сбрасывается.
func (s *Ledger) SetBudget(ctx context.Context, budget domain.Budget) error {
	if err := budget.Validate(); err != nil {
		return err
	}
	if err := s.budgets.Save(ctx, budget); err != nil {
		return err
	}
	if s.cache == nil {
		return nil
	}
	if err := s.cache.Del(ctx, budgetsCacheKey); err != nil {
		log.Printf("не удалось сбросить кеш бюджетов: %v", err)
	}
	return nil
}

// ListBudgets возвращает бюджеты. Повторный вызов в течение TTL читает кеш.
func (s *Ledger) ListBudgets(ctx context.Context) ([]domain.Budget, error) {
	if raw, ok := s.readCache(ctx, budgetsCacheKey); ok {
		var cached []domain.Budget
		if err := json.Unmarshal([]byte(raw), &cached); err == nil {
			log.Printf("кеш бюджетов: попадание, ключ %s", budgetsCacheKey)
			return cached, nil
		}
		log.Printf("кеш бюджетов повреждён, читаю базу")
	}

	items, err := s.budgets.List(ctx)
	if err != nil {
		return nil, err
	}
	s.writeCache(ctx, budgetsCacheKey, items, budgetsCacheTTL, "кеш бюджетов")
	return items, nil
}

// AddTransaction проверяет поля и лимит категории, затем сохраняет расход.
// Категория без бюджета принимается без ограничения суммы.
func (s *Ledger) AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	budget, err := s.budgets.FindByCategory(ctx, tx.Category)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.Transaction{}, err
	}
	if err == nil {
		spent, err := s.expenses.SumByCategory(ctx, tx.Category)
		if err != nil {
			return domain.Transaction{}, err
		}
		if spent+tx.Amount > budget.Limit {
			return domain.Transaction{}, domain.ErrBudgetExceeded
		}
	}
	return s.expenses.Add(ctx, tx)
}

// ListTransactions возвращает сохранённые расходы.
func (s *Ledger) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	return s.expenses.List(ctx)
}

// GetReportSummary считает суммы по категориям. Повтор с теми же датами берётся из кеша.
func (s *Ledger) GetReportSummary(ctx context.Context, from, to time.Time) ([]domain.ReportRow, error) {
	key := fmt.Sprintf("report:summary:%s:%s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	if raw, ok := s.readCache(ctx, key); ok {
		var cached []domain.ReportRow
		if err := json.Unmarshal([]byte(raw), &cached); err == nil {
			log.Printf("кеш отчёта: попадание, ключ %s", key)
			return cached, nil
		}
		log.Printf("кеш отчёта повреждён, считаю по базе")
	}

	rows, err := s.expenses.Summary(ctx, from, to)
	if err != nil {
		return nil, err
	}
	s.writeCache(ctx, key, rows, reportCacheTTL, "кеш отчёта")
	return rows, nil
}

func (s *Ledger) readCache(ctx context.Context, key string) (string, bool) {
	if s.cache == nil {
		return "", false
	}
	raw, ok, err := s.cache.Get(ctx, key)
	if err != nil {
		log.Printf("кеш %s недоступен, читаю базу: %v", key, err)
		return "", false
	}
	return raw, ok
}

func (s *Ledger) writeCache(ctx context.Context, key string, value any, ttl time.Duration, title string) {
	if s.cache == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err != nil {
		log.Printf("не удалось сериализовать %s: %v", title, err)
		return
	}
	if err := s.cache.Set(ctx, key, string(payload), ttl); err != nil {
		log.Printf("не удалось записать %s: %v", title, err)
		return
	}
	log.Printf("%s: промах, ключ %s", title, key)
}
