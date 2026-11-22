package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"ledger"
)

// API вызывает методы сервиса Ledger и не знает, где хранятся данные.
type API struct {
	svc ledger.Service
}

// New передаёт готовый сервис в HTTP-обработчики.
func New(svc ledger.Service) *API {
	return &API{svc: svc}
}

// Register подключает прежние маршруты бюджетов, транзакций и отчёта.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/transactions", a.handleTransactions)
	mux.HandleFunc("/api/budgets", a.handleBudgets)
	mux.HandleFunc("/api/reports/summary", a.handleReport)
	mux.HandleFunc("/", a.handleNotFound)
}

func (a *API) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.createTransaction(w, r)
	case http.MethodGet:
		a.listTransactions(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handleBudgets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.createBudget(w, r)
	case http.MethodGet:
		a.listBudgets(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
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

	stored, err := a.svc.AddTransaction(r.Context(), tx)
	if err != nil {
		if errors.Is(err, ledger.ErrBudgetExceeded) {
			writeError(w, http.StatusConflict, "budget exceeded")
			return
		}
		writeAPIError(w, err, "не удалось сохранить транзакцию")
		return
	}

	writeJSON(w, http.StatusCreated, TransactionToResponse(stored))
}

func (a *API) listTransactions(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListTransactions(r.Context())
	if err != nil {
		writeAPIError(w, err, "не удалось получить транзакции")
		return
	}
	response := make([]TransactionResponse, 0, len(items))
	for _, tx := range items {
		response = append(response, TransactionToResponse(tx))
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *API) createBudget(w http.ResponseWriter, r *http.Request) {
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
	if err := a.svc.SetBudget(r.Context(), budget); err != nil {
		writeAPIError(w, err, "не удалось сохранить бюджет")
		return
	}

	writeJSON(w, http.StatusCreated, BudgetToResponse(budget))
}

func (a *API) listBudgets(w http.ResponseWriter, r *http.Request) {
	items, err := a.svc.ListBudgets(r.Context())
	if err != nil {
		writeAPIError(w, err, "не удалось получить бюджеты")
		return
	}
	response := make([]BudgetResponse, 0, len(items))
	for _, budget := range items {
		response = append(response, BudgetToResponse(budget))
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *API) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "from must be YYYY-MM-DD")
		return
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "to must be YYYY-MM-DD")
		return
	}
	if to.Before(from) {
		writeError(w, http.StatusBadRequest, "to is before from")
		return
	}

	summary, err := a.svc.GetReportSummary(r.Context(), from, to)
	if err != nil {
		writeAPIError(w, err, "не удалось построить отчёт")
		return
	}
	if summary == nil {
		summary = map[string]float64{}
	}
	writeJSON(w, http.StatusOK, summary)
}

func writeAPIError(w http.ResponseWriter, err error, message string) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		writeError(w, http.StatusGatewayTimeout, "request timeout")
		return
	}
	log.Printf("%s: %v", message, err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
