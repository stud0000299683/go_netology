package api

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// DefaultRequestTimeout — предел обработки одного HTTP-запроса, если окружение его не переопределяет.
const DefaultRequestTimeout = 2 * time.Second

// WithTimeout ограничивает каждый запрос контекстом с таймаутом.
// Если обработчик не успел ответить, клиент получает 504 и JSON {"error":"request timeout"}.
func WithTimeout(limit time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), limit)
			defer cancel()

			done := make(chan struct{})
			writer := &timeoutWriter{ResponseWriter: w, header: w.Header().Clone()}
			go func() {
				next.ServeHTTP(writer, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				writer.flush()
			case <-ctx.Done():
				writer.timeout()
				<-done
			}
		})
	}
}

type timeoutWriter struct {
	http.ResponseWriter
	mu       sync.Mutex
	header   http.Header
	code     int
	body     []byte
	wrote    bool
	timedOut bool
}

func (w *timeoutWriter) Header() http.Header {
	return w.header
}

func (w *timeoutWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut || w.wrote {
		return
	}
	w.code = status
}

func (w *timeoutWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return len(p), nil
	}
	w.wrote = true
	if w.code == 0 {
		w.code = http.StatusOK
	}
	w.body = append(w.body, p...)
	return len(p), nil
}

func (w *timeoutWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return
	}
	w.writeLocked()
}

func (w *timeoutWriter) timeout() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wrote {
		w.writeLocked()
		return
	}
	w.timedOut = true
	w.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.ResponseWriter.WriteHeader(http.StatusGatewayTimeout)
	_, _ = w.ResponseWriter.Write([]byte("{\"error\":\"request timeout\"}\n"))
}

func (w *timeoutWriter) writeLocked() {
	header := w.ResponseWriter.Header()
	for key, values := range w.header {
		header.Del(key)
		for _, value := range values {
			header.Add(key, value)
		}
	}
	if w.code == 0 {
		w.code = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(w.code)
	if len(w.body) > 0 {
		_, _ = w.ResponseWriter.Write(w.body)
	}
}
