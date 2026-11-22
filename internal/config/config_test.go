package config_test

import (
	"errors"
	"testing"

	"github.com/kramirez/rex-stub/internal/config"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{"API_KEY": "k"}))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if cfg.Port != config.DefaultPort {
		t.Fatalf("puerto por defecto esperado %s, fue %s", config.DefaultPort, cfg.Port)
	}
	if cfg.Simulation.LatencyMs != 0 || cfg.Simulation.FailureRate != 0 || cfg.Simulation.ForceError {
		t.Fatalf("la simulación debía partir apagada: %+v", cfg.Simulation)
	}
}

func TestLoadFailsWithoutAPIKey(t *testing.T) {
	_, err := config.Load(env(map[string]string{"API_KEY": "  "}))

	if !errors.Is(err, config.ErrMissingAPIKey) {
		t.Fatalf("se esperaba ErrMissingAPIKey, fue %v", err)
	}
}

func TestLoadReadsSimulationSettings(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{
		"API_KEY":          "k",
		"PORT":             "9000",
		"SIM_LATENCY_MS":   "250",
		"SIM_FAILURE_RATE": "0.3",
		"SIM_FORCE_ERROR":  "true",
	}))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if cfg.Port != "9000" || cfg.Simulation.LatencyMs != 250 || cfg.Simulation.FailureRate != 0.3 || !cfg.Simulation.ForceError {
		t.Fatalf("configuración inesperada: %+v", cfg)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	cases := map[string]map[string]string{
		"latencia no numérica": {"API_KEY": "k", "SIM_LATENCY_MS": "abc"},
		"tasa fuera de rango":  {"API_KEY": "k", "SIM_FAILURE_RATE": "2"},
		"booleano inválido":    {"API_KEY": "k", "SIM_FORCE_ERROR": "quizás"},
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := config.Load(env(values)); err == nil {
				t.Fatal("se esperaba un error")
			}
		})
	}
}
