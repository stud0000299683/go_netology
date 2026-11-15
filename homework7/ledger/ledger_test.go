package ledger

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTransactionValidate(t *testing.T) {
	tests := []struct {
		name    string
		tx      Transaction
		wantErr bool
	}{
		{name: "valid", tx: Transaction{Amount: 450, Category: "еда", Description: "ланч"}, wantErr: false},
		{name: "zero amount", tx: Transaction{Amount: 0, Category: "еда"}, wantErr: true},
		{name: "negative amount", tx: Transaction{Amount: -10, Category: "еда"}, wantErr: true},
		{name: "empty category", tx: Transaction{Amount: 100, Category: ""}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertError(t, tc.tx.Validate(), tc.wantErr)
		})
	}
}

func TestBudgetValidate(t *testing.T) {
	tests := []struct {
		name    string
		budget  Budget
		wantErr bool
	}{
		{name: "positive limit", budget: Budget{Category: "еда", Limit: 5000}, wantErr: false},
		{name: "zero limit", budget: Budget{Category: "еда", Limit: 0}, wantErr: true},
		{name: "negative limit", budget: Budget{Category: "еда", Limit: -1}, wantErr: true},
		{name: "empty category", budget: Budget{Category: "", Limit: 5000}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertError(t, tc.budget.Validate(), tc.wantErr)
		})
	}
}

func TestCheckValid(t *testing.T) {
	t.Parallel()

	if err := CheckValid(Transaction{Amount: 100, Category: "еда"}); err != nil {
		t.Fatalf("корректная транзакция: %v", err)
	}
	if err := CheckValid(Budget{Category: "", Limit: 10}); err == nil {
		t.Fatal("ожидалась ошибка бюджета")
	}
}

func TestBudgetLimit(t *testing.T) {
	if err := Connect(context.Background()); err != nil {
		t.Skipf("нет подключения к PostgreSQL или Redis: %v", err)
	}
	t.Cleanup(func() {
		Reset()
		Close()
	})
	Reset()

	if err := SetBudget(Budget{Category: "еда", Limit: 5000}); err != nil {
		t.Fatalf("сохранение бюджета: %v", err)
	}
	if err := SetBudget(Budget{Category: "transport", Limit: 1500, Period: "month"}); err != nil {
		t.Fatalf("сохранение бюджета transport: %v", err)
	}

	budgets, err := ListBudgets()
	if err != nil {
		t.Fatalf("список бюджетов: %v", err)
	}
	if len(budgets) != 2 || budgets[0].Category != "transport" || budgets[1].Category != "еда" || budgets[1].Limit != 5000 {
		t.Fatalf("бюджеты = %+v", budgets)
	}

	if _, err := AddTransaction(Transaction{Amount: 0, Category: "еда"}); err == nil {
		t.Fatal("нулевая сумма должна быть отклонена")
	}

	day := time.Date(2025, time.September, 10, 0, 0, 0, 0, time.UTC)
	stored, err := AddTransaction(Transaction{
		Amount:      450,
		Category:    "еда",
		Description: "ланч",
		Date:        day,
	})
	if err != nil {
		t.Fatalf("трата в пределах лимита: %v", err)
	}
	if stored.ID == 0 {
		t.Fatal("ожидался идентификатор из базы")
	}

	before, err := ListTransactions()
	if err != nil {
		t.Fatalf("список транзакций: %v", err)
	}
	_, err = AddTransaction(Transaction{
		Amount:      4600,
		Category:    "еда",
		Description: "ужин",
		Date:        day.AddDate(0, 0, 1),
	})
	if !errors.Is(err, ErrBudgetExceeded) || err.Error() != "budget exceeded" {
		t.Fatalf("ожидалось превышение бюджета, получено: %v", err)
	}
	after, err := ListTransactions()
	if err != nil {
		t.Fatalf("список после отказа: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("число транзакций = %d, ожидалось %d", len(after), len(before))
	}

	rows, err := GetReportSummary(context.Background(), day, day)
	if err != nil {
		t.Fatalf("отчёт: %v", err)
	}
	if len(rows) != 1 || rows[0].Category != "еда" || rows[0].Total != 450 {
		t.Fatalf("отчёт = %+v", rows)
	}
	cached, err := GetReportSummary(context.Background(), day, day)
	if err != nil {
		t.Fatalf("повторный отчёт: %v", err)
	}
	if len(cached) != 1 || cached[0].Total != 450 {
		t.Fatalf("кешированный отчёт = %+v", cached)
	}
}

func assertError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if wantErr && err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if !wantErr && err != nil {
		t.Fatalf("ожидался nil, получено %v", err)
	}
}
