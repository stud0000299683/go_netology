package ledger

import (
	"errors"
	"sort"
)

// Budget is a fixed spending limit for one category.
// Period describes the budget window, such as "month"; the limit itself is per category.
type Budget struct {
	Category string
	Limit    float64
	Period   string
}

// Validate checks that the budget can be stored.
func (b Budget) Validate() error {
	if b.Limit <= 0 {
		return errors.New("budget limit must be positive")
	}
	if b.Category == "" {
		return errors.New("budget category must not be empty")
	}
	return nil
}

// SetBudget validates b and inserts or replaces the budget for b.Category.
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	budgets[b.Category] = b
	return nil
}

// ListBudgets returns a copy of all budgets sorted by category.
func ListBudgets() []Budget {
	mu.Lock()
	defer mu.Unlock()

	result := make([]Budget, 0, len(budgets))
	for _, b := range budgets {
		result = append(result, b)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Category < result[j].Category
	})
	return result
}
