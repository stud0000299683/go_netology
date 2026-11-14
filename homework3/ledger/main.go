package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Ledger service started")
	initStore()

	file, err := os.Open("budgets.json")
	if err != nil {
		fmt.Printf("failed to open budgets.json: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := LoadBudgets(bufio.NewReader(file)); err != nil {
		fmt.Printf("failed to load budgets: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Budgets loaded from budgets.json:")
	printBudgets()

	SetBudget(Budget{Category: "transport", Limit: 1500, Period: "month"})
	SetBudget(Budget{Category: "utilities", Limit: 1000, Period: "month"})
	fmt.Println("Budgets after SetBudget:")
	printBudgets()

	samples := []Transaction{
		{
			Amount:      1200,
			Category:    "еда",
			Description: "продукты",
			Date:        time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      3500,
			Category:    "еда",
			Description: "кафе",
			Date:        time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      500,
			Category:    "еда",
			Description: "ресторан",
			Date:        time.Date(2026, time.November, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      400,
			Category:    "transport",
			Description: "метро",
			Date:        time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      1200,
			Category:    "transport",
			Description: "такси",
			Date:        time.Date(2026, time.November, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      300,
			Category:    "utilities",
			Description: "электричество",
			Date:        time.Date(2026, time.November, 6, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      800,
			Category:    "utilities",
			Description: "отопление",
			Date:        time.Date(2026, time.November, 7, 0, 0, 0, 0, time.UTC),
		},
		{
			Amount:      0,
			Category:    "еда",
			Description: "пустая операция",
			Date:        time.Date(2026, time.November, 8, 0, 0, 0, 0, time.UTC),
		},
	}

	fmt.Println("Adding transactions:")
	for _, tx := range samples {
		if err := AddTransaction(tx); err != nil {
			fmt.Printf("rejected %q amount=%.2f category=%s: %v\n", tx.Description, tx.Amount, tx.Category, err)
			continue
		}
		fmt.Printf("added %q amount=%.2f category=%s\n", tx.Description, tx.Amount, tx.Category)
	}

	fmt.Println("Transactions:")
	for _, tx := range ListTransactions() {
		fmt.Printf("ID=%d Amount=%.2f Category=%s Description=%s Date=%s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date.Format("2006-01-02"))
	}
}

func printBudgets() {
	for _, b := range ListBudgets() {
		fmt.Printf("Category=%s Limit=%.2f Period=%s\n", b.Category, b.Limit, b.Period)
	}
}
