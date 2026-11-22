package service

import (
	"context"
	"errors"
	"sort"
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

type memBudgets struct {
	items map[string]domain.Budget
}

func newMemBudgets() *memBudgets {
	return &memBudgets{items: map[string]domain.Budget{}}
}

func (m *memBudgets) Save(_ context.Context, budget domain.Budget) error {
	m.items[budget.Category] = budget
	return nil
}

func (m *memBudgets) FindByCategory(_ context.Context, category string) (domain.Budget, error) {
	budget, ok := m.items[category]
	if !ok {
		return domain.Budget{}, domain.ErrNotFound
	}
	return budget, nil
}

func (m *memBudgets) List(context.Context) ([]domain.Budget, error) {
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
	items  []domain.Transaction
	nextID int
}

func newMemExpenses() *memExpenses {
	return &memExpenses{}
}

func (m *memExpenses) Add(_ context.Context, tx domain.Transaction) (domain.Transaction, error) {
	m.nextID++
	tx.ID = m.nextID
	m.items = append(m.items, tx)
	return tx, nil
}

func (m *memExpenses) List(context.Context) ([]domain.Transaction, error) {
	result := make([]domain.Transaction, len(m.items))
	copy(result, m.items)
	return result, nil
}

func (m *memExpenses) SumByCategory(_ context.Context, category string) (float64, error) {
	var sum float64
	for _, item := range m.items {
		if item.Category == category {
			sum += item.Amount
		}
	}
	return sum, nil
}

func (m *memExpenses) Categories(context.Context) ([]string, error) {
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
	var sum float64
	for _, item := range m.items {
		if item.Category != category || item.Date.Before(from) || item.Date.After(to) {
			continue
		}
		sum += item.Amount
	}
	return sum, nil
}
