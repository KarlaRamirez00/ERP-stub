package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kramirez/rex-stub/internal/simulation"
)

const DefaultPort = "8090"

var ErrMissingAPIKey = errors.New("la variable API_KEY es obligatoria")

type Config struct {
	Port       string
	APIKey     string
	Simulation simulation.Settings
}

func Load(getenv func(string) string) (Config, error) {
	apiKey := strings.TrimSpace(getenv("API_KEY"))
	if apiKey == "" {
		return Config{}, ErrMissingAPIKey
	}

	latency, err := intEnv(getenv, "SIM_LATENCY_MS")
	if err != nil {
		return Config{}, err
	}
	failureRate, err := floatEnv(getenv, "SIM_FAILURE_RATE")
	if err != nil {
		return Config{}, err
	}
	forceError, err := boolEnv(getenv, "SIM_FORCE_ERROR")
	if err != nil {
		return Config{}, err
	}

	settings := simulation.Settings{LatencyMs: latency, FailureRate: failureRate, ForceError: forceError}
	if fields := settings.Validate(); fields != nil {
		return Config{}, fmt.Errorf("configuración de simulación inválida: %v", fields)
	}

	return Config{Port: portOrDefault(getenv("PORT")), APIKey: apiKey, Simulation: settings}, nil
}

func PortOrDefault(getenv func(string) string) string {
	return portOrDefault(getenv("PORT"))
}

func portOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return DefaultPort
	}
	return strings.TrimSpace(value)
}

func intEnv(getenv func(string) string, key string) (int, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s debe ser un entero: %w", key, err)
	}
	return parsed, nil
}

func floatEnv(getenv func(string) string, key string) (float64, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s debe ser un número: %w", key, err)
	}
	return parsed, nil
}

func boolEnv(getenv func(string) string, key string) (bool, error) {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s debe ser true o false: %w", key, err)
	}
	return parsed, nil
}
