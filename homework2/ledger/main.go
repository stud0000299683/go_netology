package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Ledger service started")
	initStore()

	samples := []Transaction{
		{
			Amount:      1250.50,
			Category:    "food",
			Description: "grocery shopping",
			Date:        time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      89.99,
			Category:    "transport",
			Description: "monthly transit pass",
			Date:        time.Date(2026, time.October, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      420,
			Category:    "utilities",
			Description: "electricity bill",
			Date:        time.Date(2026, time.October, 3, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tx := range samples {
		if err := AddTransaction(tx); err != nil {
			fmt.Printf("failed to add transaction %q: %v\n", tx.Description, err)
			continue
		}
	}

	fmt.Println("Transactions:")
	for _, tx := range ListTransactions() {
		fmt.Printf("ID=%d Amount=%.2f Category=%s Description=%s Date=%s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date.Format("2006-01-02"))
	}
}
