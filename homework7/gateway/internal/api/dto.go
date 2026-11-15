package api

import (
	"errors"
	"time"

	"ledger"
)

// CreateTransactionRequest is the JSON body for creating a transaction.
type CreateTransactionRequest struct {
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

// TransactionResponse is the public JSON view of a ledger transaction.
type TransactionResponse struct {
	ID          int     `json:"id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

// CreateBudgetRequest is the JSON body for creating or updating a budget.
type CreateBudgetRequest struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

// BudgetResponse is the public JSON view of a ledger budget.
type BudgetResponse struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period,omitempty"`
}

// ReportResponse — сумма расходов категории за выбранный интервал.
type ReportResponse struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// ToTransaction maps the HTTP request onto a domain transaction.
// Date is accepted as YYYY-MM-DD.
func (r CreateTransactionRequest) ToTransaction() (ledger.Transaction, error) {
	if r.Date == "" {
		return ledger.Transaction{}, errors.New("transaction date is required")
	}
	parsed, err := time.Parse("2006-01-02", r.Date)
	if err != nil {
		return ledger.Transaction{}, errors.New("transaction date must be YYYY-MM-DD")
	}
	return ledger.Transaction{
		Amount:      r.Amount,
		Category:    r.Category,
		Description: r.Description,
		Date:        parsed,
	}, nil
}

// TransactionToResponse maps a domain transaction onto the HTTP response.
func TransactionToResponse(tx ledger.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Category:    tx.Category,
		Description: tx.Description,
		Date:        tx.Date.Format("2006-01-02"),
	}
}

// ToBudget maps the HTTP request onto a domain budget.
func (r CreateBudgetRequest) ToBudget() ledger.Budget {
	return ledger.Budget{
		Category: r.Category,
		Limit:    r.Limit,
	}
}

// BudgetToResponse maps a domain budget onto the HTTP response.
func BudgetToResponse(b ledger.Budget) BudgetResponse {
	return BudgetResponse{
		Category: b.Category,
		Limit:    b.Limit,
		Period:   b.Period,
	}
}
