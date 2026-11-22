package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ledger"
)

func TestBudgetsAPI(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mux := newMux(newFake())

		rec := perform(t, mux, http.MethodPost, "/api/budgets", `{"category":"еда","limit":5000}`)
		assertStatus(t, rec, http.StatusCreated)
		assertJSONContentType(t, rec)

		var created BudgetResponse
		decodeJSON(t, rec, &created)
		if created.Category != "еда" || created.Limit != 5000 {
			t.Fatalf("created budget = %+v", created)
		}

		listed := perform(t, mux, http.MethodGet, "/api/budgets", "")
		assertStatus(t, listed, http.StatusOK)
		assertJSONContentType(t, listed)

		var budgets []BudgetResponse
		decodeJSON(t, listed, &budgets)
		if len(budgets) != 1 || budgets[0].Category != "еда" || budgets[0].Limit != 5000 {
			t.Fatalf("budgets = %+v", budgets)
		}
	})

	t.Run("bad_json", func(t *testing.T) {
		rec := perform(t, newMux(newFake()), http.MethodPost, "/api/budgets", `{`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "invalid json")
	})

	t.Run("non_positive_limit", func(t *testing.T) {
		rec := perform(t, newMux(newFake()), http.MethodPost, "/api/budgets", `{"category":"еда","limit":0}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		if got := errorMessage(t, rec); got == "" {
			t.Fatal("expected error message")
		}
	})
}

func TestTransactionsAPI(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		mux := newMux(newFake())

		budget := perform(t, mux, http.MethodPost, "/api/budgets", `{"category":"еда","limit":5000}`)
		assertStatus(t, budget, http.StatusCreated)

		body := `{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}`
		rec := perform(t, mux, http.MethodPost, "/api/transactions", body)
		assertStatus(t, rec, http.StatusCreated)
		assertJSONContentType(t, rec)

		var created TransactionResponse
		decodeJSON(t, rec, &created)
		if created.ID == 0 || created.Amount != 450 || created.Category != "еда" || created.Description != "ланч" || created.Date != "2025-09-10" {
			t.Fatalf("created transaction = %+v", created)
		}

		listed := perform(t, mux, http.MethodGet, "/api/transactions", "")
		assertStatus(t, listed, http.StatusOK)
		assertJSONContentType(t, listed)

		var transactions []TransactionResponse
		decodeJSON(t, listed, &transactions)
		if len(transactions) != 1 || transactions[0].ID != created.ID || transactions[0].Description != "ланч" {
			t.Fatalf("transactions = %+v", transactions)
		}
	})

	t.Run("exceeded", func(t *testing.T) {
		mux := newMux(newFake())

		budget := perform(t, mux, http.MethodPost, "/api/budgets", `{"category":"еда","limit":500}`)
		assertStatus(t, budget, http.StatusCreated)
		first := perform(t, mux, http.MethodPost, "/api/transactions", `{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}`)
		assertStatus(t, first, http.StatusCreated)

		rec := perform(t, mux, http.MethodPost, "/api/transactions", `{"amount":100,"category":"еда","description":"ужин","date":"2025-09-11"}`)
		assertStatus(t, rec, http.StatusConflict)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "budget exceeded")

		listed := perform(t, mux, http.MethodGet, "/api/transactions", "")
		var transactions []TransactionResponse
		decodeJSON(t, listed, &transactions)
		if len(transactions) != 1 {
			t.Fatalf("transactions = %+v", transactions)
		}
	})

	t.Run("invalid_amount", func(t *testing.T) {
		rec := perform(t, newMux(newFake()), http.MethodPost, "/api/transactions", `{"amount":0,"category":"еда","description":"пусто","date":"2025-09-10"}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "transaction amount must be positive")
	})

	t.Run("bad_json", func(t *testing.T) {
		rec := perform(t, newMux(newFake()), http.MethodPost, "/api/transactions", `not-json`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "invalid json")
	})

	t.Run("missing_date", func(t *testing.T) {
		rec := perform(t, newMux(newFake()), http.MethodPost, "/api/transactions", `{"amount":450,"category":"еда","description":"ланч"}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "transaction date is required")
	})

	t.Run("bad_date", func(t *testing.T) {
		rec := perform(t, newMux(newFake()), http.MethodPost, "/api/transactions", `{"amount":450,"category":"еда","description":"ланч","date":"10-09-2025"}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "transaction date must be YYYY-MM-DD")
	})
}

func TestReportAPI(t *testing.T) {
	mux := newMux(newFake())

	budget := perform(t, mux, http.MethodPost, "/api/budgets", `{"category":"еда","limit":5000}`)
	assertStatus(t, budget, http.StatusCreated)
	created := perform(t, mux, http.MethodPost, "/api/transactions", `{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}`)
	assertStatus(t, created, http.StatusCreated)

	path := "/api/reports/summary?from=2025-09-01&to=2025-09-30"
	first := perform(t, mux, http.MethodGet, path, "")
	assertStatus(t, first, http.StatusOK)
	assertJSONContentType(t, first)
	second := perform(t, mux, http.MethodGet, path, "")
	assertStatus(t, second, http.StatusOK)

	var rows map[string]float64
	decodeJSON(t, second, &rows)
	if len(rows) != 1 || rows["еда"] != 450 {
		t.Fatalf("report = %+v", rows)
	}
}

func TestPing(t *testing.T) {
	mux := newMux(newFake())
	rec := perform(t, mux, http.MethodGet, "/ping", "")
	assertStatus(t, rec, http.StatusOK)
	if rec.Body.String() != "pong" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestRequestTimeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/reports/summary", func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		writeAPIError(w, r.Context().Err(), "не удалось построить отчёт")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/reports/summary?from=2025-09-01&to=2025-09-30", nil)
	rec := httptest.NewRecorder()
	WithTimeout(20*time.Millisecond)(mux).ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusGatewayTimeout)
	assertJSONContentType(t, rec)
	assertErrorMessage(t, rec, "request timeout")
}

func TestBulkImportAPI(t *testing.T) {
	f := newFake()
	mux := newMux(f)
	perform(t, mux, http.MethodPost, "/api/budgets", `{"category":"еда","limit":5000}`)

	body := `[
		{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"},
		{"amount":0,"category":"еда","description":"пусто","date":"2025-09-10"},
		{"amount":10,"category":"еда","description":"дата","date":"вчера"}
	]`
	rec := perform(t, mux, http.MethodPost, "/api/transactions/bulk?workers=2", body)
	assertStatus(t, rec, http.StatusOK)
	assertJSONContentType(t, rec)

	var summary BulkImportResponse
	decodeJSON(t, rec, &summary)
	if summary.Accepted != 1 || summary.Rejected != 2 || len(summary.Errors) != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.Errors[0].Index != 1 || summary.Errors[1].Index != 2 {
		t.Fatalf("errors = %+v", summary.Errors)
	}
	if f.lastWorkers != 2 {
		t.Fatalf("workers = %d", f.lastWorkers)
	}

	defaultWorkers := perform(t, mux, http.MethodPost, "/api/transactions/bulk", `[{"amount":10,"category":"транспорт","description":"автобус","date":"2025-09-11"}]`)
	assertStatus(t, defaultWorkers, http.StatusOK)
	if f.lastWorkers != ledger.DefaultImportWorkers {
		t.Fatalf("workers = %d, want %d", f.lastWorkers, ledger.DefaultImportWorkers)
	}

	badWorkers := perform(t, mux, http.MethodPost, "/api/transactions/bulk?workers=0", `[]`)
	assertStatus(t, badWorkers, http.StatusBadRequest)
	assertErrorMessage(t, badWorkers, "workers must be a positive integer")

	badJSON := perform(t, mux, http.MethodPost, "/api/transactions/bulk", `{`)
	assertStatus(t, badJSON, http.StatusBadRequest)
	assertErrorMessage(t, badJSON, "invalid json")

	method := perform(t, mux, http.MethodGet, "/api/transactions/bulk", "")
	assertStatus(t, method, http.StatusMethodNotAllowed)

	req := httptest.NewRequest(http.MethodPost, "/api/transactions/bulk", strings.NewReader(`[{"amount":10,"category":"еда","description":"x","date":"2025-09-10"}]`))
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req.WithContext(ctx))
	assertStatus(t, rec, http.StatusGatewayTimeout)
	assertErrorMessage(t, rec, "request timeout")
}

func TestRoutes(t *testing.T) {
	mux := newMux(newFake())

	missing := perform(t, mux, http.MethodGet, "/missing", "")
	assertStatus(t, missing, http.StatusNotFound)
	assertJSONContentType(t, missing)
	assertErrorMessage(t, missing, "not found")

	method := perform(t, mux, http.MethodPut, "/api/transactions", "")
	assertStatus(t, method, http.StatusMethodNotAllowed)
	assertJSONContentType(t, method)
	assertErrorMessage(t, method, "method not allowed")

	budgetMethod := perform(t, mux, http.MethodPut, "/api/budgets", "")
	assertStatus(t, budgetMethod, http.StatusMethodNotAllowed)
	assertJSONContentType(t, budgetMethod)
	assertErrorMessage(t, budgetMethod, "method not allowed")
}

func newMux(svc ledger.Service) *http.ServeMux {
	mux := http.NewServeMux()
	New(svc).Register(mux)
	return mux
}

func perform(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	Logging(handler).ServeHTTP(rec, req)
	return rec
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, want, rec.Body.String())
	}
}

func assertJSONContentType(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	const want = "application/json; charset=utf-8"
	if got := rec.Header().Get("Content-Type"); got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}

func assertErrorMessage(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	if got := errorMessage(t, rec); got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	decodeJSON(t, rec, &payload)
	return payload.Error
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode json: %v, body = %s", err, rec.Body.String())
	}
}

// fake повторяет правила сервиса в памяти, чтобы проверить HTTP без базы.
type fake struct {
	budgets     map[string]ledger.Budget
	txs         []ledger.Transaction
	nextID      int
	lastWorkers int
}

func newFake() *fake {
	return &fake{budgets: map[string]ledger.Budget{}}
}

func (f *fake) SetBudget(_ context.Context, budget ledger.Budget) error {
	if err := budget.Validate(); err != nil {
		return err
	}
	f.budgets[budget.Category] = budget
	return nil
}

func (f *fake) ListBudgets(context.Context) ([]ledger.Budget, error) {
	result := make([]ledger.Budget, 0, len(f.budgets))
	for _, item := range f.budgets {
		result = append(result, item)
	}
	return result, nil
}

func (f *fake) AddTransaction(_ context.Context, tx ledger.Transaction) (ledger.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return ledger.Transaction{}, err
	}
	if budget, ok := f.budgets[tx.Category]; ok {
		var spent float64
		for _, item := range f.txs {
			if item.Category == tx.Category {
				spent += item.Amount
			}
		}
		if spent+tx.Amount > budget.Limit {
			return ledger.Transaction{}, ledger.ErrBudgetExceeded
		}
	}
	f.nextID++
	tx.ID = f.nextID
	f.txs = append(f.txs, tx)
	return tx, nil
}

func (f *fake) ListTransactions(context.Context) ([]ledger.Transaction, error) {
	result := make([]ledger.Transaction, len(f.txs))
	copy(result, f.txs)
	return result, nil
}

func (f *fake) ImportTransactions(ctx context.Context, items []ledger.IndexedTransaction, workers int) (ledger.ImportSummary, error) {
	if err := ctx.Err(); err != nil {
		return ledger.ImportSummary{Errors: []ledger.ImportError{}}, err
	}
	f.lastWorkers = workers
	summary := ledger.ImportSummary{Errors: []ledger.ImportError{}}
	for _, item := range items {
		if _, err := f.AddTransaction(ctx, item.Transaction); err != nil {
			summary.Rejected++
			summary.Errors = append(summary.Errors, ledger.ImportError{Index: item.Index, Error: err.Error()})
			continue
		}
		summary.Accepted++
	}
	return summary, nil
}

func (f *fake) GetReportSummary(ctx context.Context, from, to time.Time) (map[string]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	totals := map[string]float64{}
	for _, tx := range f.txs {
		if tx.Date.Before(from) || tx.Date.After(to) {
			continue
		}
		totals[tx.Category] += tx.Amount
	}
	return totals, nil
}
