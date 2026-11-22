package service

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"ledger/internal/domain"
)

func TestBudgetLimit(t *testing.T) {
	budgets := newMemBudgets()
	expenses := newMemExpenses()
	svc := New(budgets, expenses, nil)
	ctx := context.Background()

	if err := svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 5000}); err != nil {
		t.Fatalf("сохранение бюджета: %v", err)
	}
	if err := svc.SetBudget(ctx, domain.Budget{Category: "transport", Limit: 1500, Period: "month"}); err != nil {
		t.Fatalf("сохранение бюджета transport: %v", err)
	}
	if err := svc.SetBudget(ctx, domain.Budget{Category: "", Limit: 100}); err == nil {
		t.Fatal("пустая категория должна быть отклонена")
	}

	got, err := svc.ListBudgets(ctx)
	if err != nil {
		t.Fatalf("список бюджетов: %v", err)
	}
	if len(got) != 2 || got[0].Category != "transport" || got[1].Category != "еда" || got[1].Limit != 5000 {
		t.Fatalf("бюджеты = %+v", got)
	}

	if _, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 0, Category: "еда"}); err == nil {
		t.Fatal("нулевая сумма должна быть отклонена")
	}

	day := time.Date(2025, time.September, 10, 0, 0, 0, 0, time.UTC)
	stored, err := svc.AddTransaction(ctx, domain.Transaction{
		Amount:      450,
		Category:    "еда",
		Description: "ланч",
		Date:        day,
	})
	if err != nil {
		t.Fatalf("трата в пределах лимита: %v", err)
	}
	if stored.ID == 0 {
		t.Fatal("ожидался идентификатор")
	}

	before, err := svc.ListTransactions(ctx)
	if err != nil {
		t.Fatalf("список транзакций: %v", err)
	}
	_, err = svc.AddTransaction(ctx, domain.Transaction{
		Amount:      4600,
		Category:    "еда",
		Description: "ужин",
		Date:        day.AddDate(0, 0, 1),
	})
	if !errors.Is(err, domain.ErrBudgetExceeded) || err.Error() != "budget exceeded" {
		t.Fatalf("ожидалось превышение бюджета, получено: %v", err)
	}
	after, err := svc.ListTransactions(ctx)
	if err != nil {
		t.Fatalf("список после отказа: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("число транзакций = %d, ожидалось %d", len(after), len(before))
	}

	summary, err := svc.GetReportSummary(ctx, day, day)
	if err != nil {
		t.Fatalf("отчёт: %v", err)
	}
	if len(summary) != 1 || summary["еда"] != 450 {
		t.Fatalf("отчёт = %+v", summary)
	}
}

func TestReportSummaryCanceled(t *testing.T) {
	expenses := &blockingExpenses{started: make(chan struct{})}
	svc := New(newMemBudgets(), expenses, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := svc.GetReportSummary(ctx, time.Now(), time.Now())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ожидалась отмена, получено: %v", err)
	}
}

func TestImportTransactions(t *testing.T) {
	budgets := newMemBudgets()
	expenses := newMemExpenses()
	svc := New(budgets, expenses, nil)
	ctx := context.Background()
	if err := svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 100}); err != nil {
		t.Fatalf("бюджет: %v", err)
	}

	day := time.Date(2025, 9, 10, 0, 0, 0, 0, time.UTC)
	items := []domain.IndexedTransaction{
		{Index: 0, Transaction: domain.Transaction{Amount: 40, Category: "еда", Date: day}},
		{Index: 1, Transaction: domain.Transaction{Amount: 0, Category: "еда", Date: day}},
		{Index: 2, Transaction: domain.Transaction{Amount: 70, Category: "еда", Date: day}},
		{Index: 3, Transaction: domain.Transaction{Amount: 10, Category: "транспорт", Date: day}},
	}
	summary, err := svc.ImportTransactions(ctx, items, 4)
	if err != nil {
		t.Fatalf("импорт: %v", err)
	}
	if summary.Accepted != 2 || summary.Rejected != 2 || len(summary.Errors) != 2 {
		t.Fatalf("сводка = %+v", summary)
	}
	if summary.Errors[0].Index > summary.Errors[1].Index {
		t.Fatalf("ошибки не упорядочены: %+v", summary.Errors)
	}
}

func TestImportTransactionsCanceled(t *testing.T) {
	svc := New(newMemBudgets(), &slowAddExpenses{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	summary, err := svc.ImportTransactions(ctx, []domain.IndexedTransaction{{
		Index:       0,
		Transaction: domain.Transaction{Amount: 10, Category: "еда", Date: time.Now()},
	}}, 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ожидалась отмена, получено: %v, сводка = %+v", err, summary)
	}
}

type slowAddExpenses struct {
	memExpenses
}

func (s *slowAddExpenses) Add(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return domain.Transaction{}, ctx.Err()
	case <-timer.C:
		return s.memExpenses.Add(ctx, tx)
	}
}

type memBudgets struct {
	mu    sync.Mutex
	items map[string]domain.Budget
}

func newMemBudgets() *memBudgets {
	return &memBudgets{items: map[string]domain.Budget{}}
}

func (m *memBudgets) Save(_ context.Context, budget domain.Budget) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[budget.Category] = budget
	return nil
}

func (m *memBudgets) FindByCategory(_ context.Context, category string) (domain.Budget, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	budget, ok := m.items[category]
	if !ok {
		return domain.Budget{}, domain.ErrNotFound
	}
	return budget, nil
}

func (m *memBudgets) List(context.Context) ([]domain.Budget, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.Budget, 0, len(m.items))
	for _, item := range m.items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Category < result[j].Category
	})
	return result, nil
}

type memExpenses struct {
	mu     sync.Mutex
	items  []domain.Transaction
	nextID int
}

func newMemExpenses() *memExpenses {
	return &memExpenses{}
}

func (m *memExpenses) Add(_ context.Context, tx domain.Transaction) (domain.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	tx.ID = m.nextID
	m.items = append(m.items, tx)
	return tx, nil
}

func (m *memExpenses) List(context.Context) ([]domain.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.Transaction, len(m.items))
	copy(result, m.items)
	return result, nil
}

func (m *memExpenses) SumByCategory(_ context.Context, category string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sum float64
	for _, item := range m.items {
		if item.Category == category {
			sum += item.Amount
		}
	}
	return sum, nil
}

func (m *memExpenses) Categories(context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[string]struct{}{}
	var categories []string
	for _, item := range m.items {
		if _, ok := seen[item.Category]; ok {
			continue
		}
		seen[item.Category] = struct{}{}
		categories = append(categories, item.Category)
	}
	return categories, nil
}

type blockingExpenses struct {
	memExpenses
	started chan struct{}
}

func (b *blockingExpenses) Categories(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{"еда"}, nil
}

func (b *blockingExpenses) SumByCategoryPeriod(ctx context.Context, category string, from, to time.Time) (float64, error) {
	select {
	case <-b.started:
	default:
		close(b.started)
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(2 * time.Second):
		return 10, nil
	}
}

func (m *memExpenses) SumByCategoryPeriod(_ context.Context, category string, from, to time.Time) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sum float64
	for _, item := range m.items {
		if item.Category != category || item.Date.Before(from) || item.Date.After(to) {
			continue
		}
		sum += item.Amount
	}
	return sum, nil
}
