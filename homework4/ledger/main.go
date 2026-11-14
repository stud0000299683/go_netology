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

	fmt.Println("CheckValid:")
	reportCheck("transaction", Transaction{
		Amount:   100,
		Category: "еда",
		Date:     time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
	})
	reportCheck("transaction", Transaction{
		Amount:   -10,
		Category: "еда",
	})
	reportCheck("budget", Budget{Category: "еда", Limit: 5000, Period: "month"})
	reportCheck("budget", Budget{Category: "transport", Limit: 0})

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

	if err := SetBudget(Budget{Category: "transport", Limit: 1500, Period: "month"}); err != nil {
		fmt.Printf("failed to set budget: %v\n", err)
		os.Exit(1)
	}
	if err := SetBudget(Budget{Category: "utilities", Limit: 1000, Period: "month"}); err != nil {
		fmt.Printf("failed to set budget: %v\n", err)
		os.Exit(1)
	}
	if err := SetBudget(Budget{Category: "", Limit: 100}); err != nil {
		fmt.Printf("rejected budget category=%q limit=%.2f: %v\n", "", 100.0, err)
	}
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
		{
			Amount:      100,
			Category:    "",
			Description: "без категории",
			Date:        time.Date(2026, time.November, 9, 0, 0, 0, 0, time.UTC),
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

func reportCheck(kind string, v Validatable) {
	if err := CheckValid(v); err != nil {
		fmt.Printf("CheckValid %s: %v\n", kind, err)
		return
	}
	fmt.Printf("CheckValid %s: ok\n", kind)
}

func printBudgets() {
	for _, b := range ListBudgets() {
		fmt.Printf("Category=%s Limit=%.2f Period=%s\n", b.Category, b.Limit, b.Period)
	}
}
