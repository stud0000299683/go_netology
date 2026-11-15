package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ledger"
)

func TestBudgetsAPI(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		resetLedger(t)
		mux := newMux()

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
		resetLedger(t)
		rec := perform(t, newMux(), http.MethodPost, "/api/budgets", `{`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "invalid json")
	})

	t.Run("non_positive_limit", func(t *testing.T) {
		resetLedger(t)
		rec := perform(t, newMux(), http.MethodPost, "/api/budgets", `{"category":"еда","limit":0}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		if got := errorMessage(t, rec); got == "" {
			t.Fatal("expected error message")
		}
	})
}

func TestTransactionsAPI(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		resetLedger(t)
		mux := newMux()

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
		resetLedger(t)
		mux := newMux()

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
		resetLedger(t)
		rec := perform(t, newMux(), http.MethodPost, "/api/transactions", `{"amount":0,"category":"еда","description":"пусто","date":"2025-09-10"}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "transaction amount must be positive")
	})

	t.Run("bad_json", func(t *testing.T) {
		resetLedger(t)
		rec := perform(t, newMux(), http.MethodPost, "/api/transactions", `not-json`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "invalid json")
	})

	t.Run("missing_date", func(t *testing.T) {
		resetLedger(t)
		rec := perform(t, newMux(), http.MethodPost, "/api/transactions", `{"amount":450,"category":"еда","description":"ланч"}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "transaction date is required")
	})

	t.Run("bad_date", func(t *testing.T) {
		resetLedger(t)
		rec := perform(t, newMux(), http.MethodPost, "/api/transactions", `{"amount":450,"category":"еда","description":"ланч","date":"10-09-2025"}`)
		assertStatus(t, rec, http.StatusBadRequest)
		assertJSONContentType(t, rec)
		assertErrorMessage(t, rec, "transaction date must be YYYY-MM-DD")
	})
}

func TestRoutes(t *testing.T) {
	resetLedger(t)
	mux := newMux()

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

func resetLedger(t *testing.T) {
	t.Helper()
	ledger.Reset()
	t.Cleanup(ledger.Reset)
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux)
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
