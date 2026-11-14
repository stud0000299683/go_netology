package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"ledger"
)

// Register attaches the transaction and budget routes to mux.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/transactions", handleTransactions)
	mux.HandleFunc("/api/budgets", handleBudgets)
	mux.HandleFunc("/", handleNotFound)
}

func handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createTransaction(w, r)
	case http.MethodGet:
		listTransactions(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleBudgets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createBudget(w, r)
	case http.MethodGet:
		listBudgets(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

func createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	tx, err := req.ToTransaction()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := tx.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	stored, err := ledger.AddTransaction(tx)
	if err != nil {
		if errors.Is(err, ledger.ErrBudgetExceeded) {
			writeError(w, http.StatusConflict, "budget exceeded")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, TransactionToResponse(stored))
}

func listTransactions(w http.ResponseWriter, r *http.Request) {
	items := ledger.ListTransactions()
	response := make([]TransactionResponse, 0, len(items))
	for _, tx := range items {
		response = append(response, TransactionToResponse(tx))
	}
	writeJSON(w, http.StatusOK, response)
}

func createBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	budget := req.ToBudget()
	if err := budget.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ledger.SetBudget(budget); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, BudgetToResponse(budget))
}

func listBudgets(w http.ResponseWriter, r *http.Request) {
	items := ledger.ListBudgets()
	response := make([]BudgetResponse, 0, len(items))
	for _, budget := range items {
		response = append(response, BudgetToResponse(budget))
	}
	writeJSON(w, http.StatusOK, response)
}
