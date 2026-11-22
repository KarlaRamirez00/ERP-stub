package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kramirez/rex-stub/internal/middleware"
	"github.com/kramirez/rex-stub/internal/simulation"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func call(handler http.Handler, apiKey string) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code
}

func TestAPIKeyRejectsMissingAndWrongKey(t *testing.T) {
	handler := middleware.APIKey("secret")(ok)

	if code := call(handler, ""); code != http.StatusUnauthorized {
		t.Fatalf("sin clave: esperado 401, fue %d", code)
	}
	if code := call(handler, "otra"); code != http.StatusUnauthorized {
		t.Fatalf("clave incorrecta: esperado 401, fue %d", code)
	}
}

func TestAPIKeyAllowsCorrectKey(t *testing.T) {
	handler := middleware.APIKey("secret")(ok)

	if code := call(handler, "secret"); code != http.StatusOK {
		t.Fatalf("esperado 200, fue %d", code)
	}
}

func TestSimulationForceErrorReturns503(t *testing.T) {
	state := simulation.NewState(simulation.Settings{ForceError: true})
	handler := middleware.Simulation(state, func() float64 { return 0.99 })(ok)

	if code := call(handler, ""); code != http.StatusServiceUnavailable {
		t.Fatalf("esperado 503, fue %d", code)
	}
}

func TestSimulationFailureRateUsesRandomValue(t *testing.T) {
	state := simulation.NewState(simulation.Settings{FailureRate: 0.5})

	failing := middleware.Simulation(state, func() float64 { return 0.1 })(ok)
	passing := middleware.Simulation(state, func() float64 { return 0.9 })(ok)

	if code := call(failing, ""); code != http.StatusServiceUnavailable {
		t.Fatalf("con azar 0.1 esperado 503, fue %d", code)
	}
	if code := call(passing, ""); code != http.StatusOK {
		t.Fatalf("con azar 0.9 esperado 200, fue %d", code)
	}
}

func TestSimulationOffPassesThrough(t *testing.T) {
	state := simulation.NewState(simulation.Settings{})
	handler := middleware.Simulation(state, func() float64 { return 0 })(ok)

	if code := call(handler, ""); code != http.StatusOK {
		t.Fatalf("esperado 200, fue %d", code)
	}
}

func TestChainRunsFirstMiddlewareOutermost(t *testing.T) {
	state := simulation.NewState(simulation.Settings{ForceError: true})
	handler := middleware.Chain(ok, middleware.APIKey("secret"), middleware.Simulation(state, func() float64 { return 0 }))

	if code := call(handler, ""); code != http.StatusUnauthorized {
		t.Fatalf("la API key debía rechazar antes que la simulación: %d", code)
	}
	if code := call(handler, "secret"); code != http.StatusServiceUnavailable {
		t.Fatalf("con clave válida debía aplicar la simulación: %d", code)
	}
}
