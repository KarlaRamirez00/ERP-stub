package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"github.com/kramirez/rex-stub/internal/httpx"
	"github.com/kramirez/rex-stub/internal/simulation"
)

const apiKeyHeader = "X-API-Key"

type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func APIKey(expected string) Middleware {
	expectedBytes := []byte(expected)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := []byte(r.Header.Get(apiKeyHeader))
			if subtle.ConstantTimeCompare(provided, expectedBytes) != 1 {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "API key ausente o inválida.", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Simulation(state *simulation.State, random func() float64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			settings := state.Get()
			if settings.LatencyMs > 0 {
				select {
				case <-time.After(time.Duration(settings.LatencyMs) * time.Millisecond):
				case <-r.Context().Done():
					return
				}
			}
			if settings.ForceError || (settings.FailureRate > 0 && random() < settings.FailureRate) {
				httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeUnavailable, "Rex+ no respondió (falla simulada).", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"durationMs", time.Since(start).Milliseconds(),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}
