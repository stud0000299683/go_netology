package ledger

import (
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
		{
			name:    "valid",
			tx:      Transaction{Amount: 450, Category: "еда", Description: "ланч"},
			wantErr: false,
		},
		{
			name:    "zero amount",
			tx:      Transaction{Amount: 0, Category: "еда"},
			wantErr: true,
		},
		{
			name:    "negative amount",
			tx:      Transaction{Amount: -10, Category: "еда"},
			wantErr: true,
		},
		{
			name:    "empty category",
			tx:      Transaction{Amount: 100, Category: ""},
			wantErr: true,
		},
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
		{
			name:    "positive limit",
			budget:  Budget{Category: "еда", Limit: 5000},
			wantErr: false,
		},
		{
			name:    "zero limit",
			budget:  Budget{Category: "еда", Limit: 0},
			wantErr: true,
		},
		{
			name:    "negative limit",
			budget:  Budget{Category: "еда", Limit: -1},
			wantErr: true,
		},
		{
			name:    "empty category",
			budget:  Budget{Category: "", Limit: 5000},
			wantErr: true,
		},
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
		t.Fatalf("valid transaction: %v", err)
	}
	if err := CheckValid(Budget{Category: "", Limit: 10}); err == nil {
		t.Fatal("expected invalid budget")
	}
}

func TestBudgetLimit(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := SetBudget(Budget{Category: "еда", Limit: 5000}); err != nil {
		t.Fatalf("set budget: %v", err)
	}
	if err := SetBudget(Budget{Category: "transport", Limit: 1500, Period: "month"}); err != nil {
		t.Fatalf("set transport budget: %v", err)
	}
	gotBudgets := ListBudgets()
	if len(gotBudgets) != 2 || gotBudgets[0].Category != "transport" || gotBudgets[1].Category != "еда" {
		t.Fatalf("budgets = %+v", gotBudgets)
	}

	if _, err := AddTransaction(Transaction{Amount: 0, Category: "еда"}); err == nil {
		t.Fatal("expected invalid transaction to be rejected")
	}

	day := time.Date(2025, time.September, 10, 0, 0, 0, 0, time.UTC)
	stored, err := AddTransaction(Transaction{
		Amount:      450,
		Category:    "еда",
		Description: "ланч",
		Date:        day,
	})
	if err != nil {
		t.Fatalf("add within limit: %v", err)
	}
	if stored.ID == 0 {
		t.Fatal("expected assigned id")
	}

	before := len(ListTransactions())
	_, err = AddTransaction(Transaction{
		Amount:      4600,
		Category:    "еда",
		Description: "ужин",
		Date:        day,
	})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected budget exceeded, got %v", err)
	}
	if err == nil || err.Error() != "budget exceeded" {
		t.Fatalf("error text = %v", err)
	}
	if got := len(ListTransactions()); got != before {
		t.Fatalf("transaction count = %d, want %d", got, before)
	}

	if err := SetBudget(Budget{Category: "", Limit: 100}); err == nil {
		t.Fatal("expected invalid budget to be rejected")
	}
	if got := len(ListBudgets()); got != 2 {
		t.Fatalf("budget count = %d, want 2", got)
	}
}

func assertError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if wantErr && err == nil {
		t.Fatal("expected error, got nil")
	}
	if !wantErr && err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
