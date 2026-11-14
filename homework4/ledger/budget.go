package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
)

// Budget is a fixed spending limit for one category.
// Period describes the budget window, such as "month"; the limit itself is per category.
type Budget struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period,omitempty"`
}

// budgets stores one budget per category. It is initialized empty when the service starts.
var budgets map[string]Budget

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
	if budgets == nil {
		budgets = make(map[string]Budget)
	}
	budgets[b.Category] = b
	return nil
}

// ListBudgets returns a copy of all budgets sorted by category.
func ListBudgets() []Budget {
	result := make([]Budget, 0, len(budgets))
	for _, b := range budgets {
		result = append(result, b)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Category < result[j].Category
	})
	return result
}

// LoadBudgets reads a JSON array of budgets and stores each one through SetBudget.
func LoadBudgets(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read budgets: %w", err)
	}

	var list []Budget
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("parse budgets: %w", err)
	}

	for _, b := range list {
		if err := b.Validate(); err != nil {
			return err
		}
	}
	for _, b := range list {
		if err := SetBudget(b); err != nil {
			return err
		}
	}
	return nil
}
