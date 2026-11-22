package domain

import "testing"

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

func assertError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if wantErr && err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if !wantErr && err != nil {
		t.Fatalf("ожидался nil, получено %v", err)
	}
}
