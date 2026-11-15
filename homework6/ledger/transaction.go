package ledger

import (
	"errors"
	"sync"
	"time"
)

// ErrBudgetExceeded is returned when a transaction would exceed its category budget.
var ErrBudgetExceeded = errors.New("budget exceeded")

// Transaction is a financial transaction stored by the Ledger service.
type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

var (
	mu           sync.Mutex
	transactions []Transaction
	budgets      map[string]Budget
)

func init() {
	transactions = make([]Transaction, 0)
	budgets = make(map[string]Budget)
}

// Reset clears in-memory transactions and budgets so tests do not share state.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	transactions = make([]Transaction, 0)
	budgets = make(map[string]Budget)
}

// Validate checks the transaction fields. Date is time.Time, so its format is already fixed by the type.
func (tx Transaction) Validate() error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}
	if tx.Category == "" {
		return errors.New("transaction category must not be empty")
	}
	return nil
}

// AddTransaction validates tx and appends it to the in-memory store.
// The returned transaction includes the assigned ID.
// If the category has a budget, the transaction is rejected when the category
// total including tx would exceed that budget's limit.
func AddTransaction(tx Transaction) (Transaction, error) {
	if err := tx.Validate(); err != nil {
		return Transaction{}, err
	}

	mu.Lock()
	defer mu.Unlock()

	if budget, ok := budgets[tx.Category]; ok {
		if categorySpent(tx.Category)+tx.Amount > budget.Limit {
			return Transaction{}, ErrBudgetExceeded
		}
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return tx, nil
}

// ListTransactions returns a copy of all stored transactions.
func ListTransactions() []Transaction {
	mu.Lock()
	defer mu.Unlock()

	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}

// categorySpent sums transactions for a category. The caller must hold mu.
func categorySpent(category string) float64 {
	var sum float64
	for _, tx := range transactions {
		if tx.Category == category {
			sum += tx.Amount
		}
	}
	return sum
}
