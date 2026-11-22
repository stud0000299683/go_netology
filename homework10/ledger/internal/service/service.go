// Пакет service применяет правила учёта и обращается к хранилищу только через интерфейсы домена.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"ledger/internal/domain"
)

const (
	budgetsCacheKey      = "budgets:all"
	budgetsCacheTTL      = 30 * time.Second
	reportCacheTTL       = 30 * time.Second
	DefaultImportWorkers = 4
	maxImportWorkers     = 32
)

// Service — операции, которые нужны HTTP-слою.
type Service interface {
	SetBudget(ctx context.Context, budget domain.Budget) error
	ListBudgets(ctx context.Context) ([]domain.Budget, error)
	AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error)
	ListTransactions(ctx context.Context) ([]domain.Transaction, error)
	GetReportSummary(ctx context.Context, from, to time.Time) (map[string]float64, error)
	ImportTransactions(ctx context.Context, items []domain.IndexedTransaction, workers int) (domain.ImportSummary, error)
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
	spendMu  sync.Map
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
// Проверка лимита и запись одной категории идут под общим замком, чтобы параллельный импорт не пропускал обе траты.
func (s *Ledger) AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return domain.Transaction{}, err
	}

	var stored domain.Transaction
	err := s.withCategoryLock(tx.Category, func() error {
		budget, err := s.budgets.FindByCategory(ctx, tx.Category)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if err == nil {
			spent, err := s.expenses.SumByCategory(ctx, tx.Category)
			if err != nil {
				return err
			}
			if spent+tx.Amount > budget.Limit {
				return domain.ErrBudgetExceeded
			}
		}
		stored, err = s.expenses.Add(ctx, tx)
		return err
	})
	if err != nil {
		return domain.Transaction{}, err
	}
	return stored, nil
}

func (s *Ledger) withCategoryLock(category string, fn func() error) error {
	value, _ := s.spendMu.LoadOrStore(category, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

// ListTransactions возвращает сохранённые расходы.
func (s *Ledger) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	return s.expenses.List(ctx)
}

// GetReportSummary считает суммы по категориям параллельно.
// Повтор с теми же датами в течение TTL берётся из кеша.
func (s *Ledger) GetReportSummary(ctx context.Context, from, to time.Time) (map[string]float64, error) {
	key := fmt.Sprintf("report:summary:%s:%s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	if raw, ok := s.readCache(ctx, key); ok {
		var cached map[string]float64
		if err := json.Unmarshal([]byte(raw), &cached); err == nil {
			log.Printf("кеш отчёта: попадание, ключ %s", key)
			if cached == nil {
				cached = map[string]float64{}
			}
			return cached, nil
		}
		log.Printf("кеш отчёта повреждён, считаю по базе")
	}

	summary := map[string]float64{}
	stop := make(chan struct{})
	var heartbeat sync.WaitGroup
	heartbeat.Add(1)
	go func() {
		defer heartbeat.Done()
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("идёт расчёт отчёта")
			case <-ctx.Done():
				log.Printf("расчёт отчёта прерван по ctx.Done(), тикер остановлен")
				return
			case <-stop:
				if ctx.Err() != nil {
					log.Printf("расчёт отчёта прерван по ctx.Done(), тикер остановлен")
					return
				}
				log.Printf("расчёт отчёта завершён, тикер остановлен")
				return
			}
		}
	}()
	defer func() {
		close(stop)
		heartbeat.Wait()
	}()

	categories, err := s.expenses.Categories(ctx)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(categories) == 0 {
		s.writeCache(ctx, key, summary, reportCacheTTL, "кеш отчёта")
		return summary, nil
	}

	// У каждой горутины свой слот, поэтому запись не пересекается.
	totals := make([]float64, len(categories))
	errs := make([]error, len(categories))
	var wg sync.WaitGroup
	for i, category := range categories {
		wg.Add(1)
		go func(i int, category string) {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				errs[i] = err
				return
			}
			total, err := s.expenses.SumByCategoryPeriod(ctx, category, from, to)
			if err != nil {
				errs[i] = err
				return
			}
			totals[i] = total
		}(i, category)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for i, category := range categories {
		if totals[i] != 0 {
			summary[category] = totals[i]
		}
	}
	s.writeCache(ctx, key, summary, reportCacheTTL, "кеш отчёта")
	return summary, nil
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
