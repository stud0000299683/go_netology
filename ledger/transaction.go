package main

import (
	"errors"
	"time"
)

// Transaction is a financial transaction stored by the Ledger service.
type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

// transactions is the in-memory store. It is initialized empty when the service starts.
var transactions []Transaction

func initStore() {
	transactions = make([]Transaction, 0)
}

// AddTransaction appends tx to the in-memory store.
// ID is assigned automatically as len(store)+1.
// A zero amount is rejected.
func AddTransaction(tx Transaction) error {
	if tx.Amount == 0 {
		return errors.New("transaction amount must not be zero")
	}

	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return nil
}

// ListTransactions returns a copy of all stored transactions.
func ListTransactions() []Transaction {
	result := make([]Transaction, len(transactions))
	copy(result, transactions)
	return result
}
