package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kramirez/rex-stub/internal/config"
	"github.com/kramirez/rex-stub/internal/rex"
	"github.com/kramirez/rex-stub/internal/server"
	"github.com/kramirez/rex-stub/internal/simulation"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
	healthTimeout     = 2 * time.Second
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "consulta /health y termina con 0 si el servicio está arriba")
	flag.Parse()

	if *healthcheck {
		os.Exit(checkHealth(config.PortOrDefault(os.Getenv)))
	}
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		logger.Error("configuración inválida", "error", err)
		return 1
	}

	handler := server.NewHandler(server.Deps{
		APIKey:  cfg.APIKey,
		State:   simulation.NewState(cfg.Simulation),
		Service: rex.NewService(rex.NewRepository()),
		Logger:  logger,
		Random:  rand.Float64,
	})
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: handler, ReadHeaderTimeout: readHeaderTimeout}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() { serverErr <- srv.ListenAndServe() }()
	logger.Info("rex-stub iniciado", "port", cfg.Port)

	select {
	case err := <-serverErr:
		logger.Error("el servidor se detuvo", "error", err)
		return 1
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("apagado con errores", "error", err)
		return 1
	}
	logger.Info("rex-stub detenido")
	return 0
}

func checkHealth(port string) int {
	client := http.Client{Timeout: healthTimeout}
	resp, err := client.Get("http://localhost:" + port + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
