package server

import (
	"log/slog"
	"net/http"

	"github.com/kramirez/rex-stub/internal/httpx"
	"github.com/kramirez/rex-stub/internal/middleware"
	"github.com/kramirez/rex-stub/internal/rex"
	"github.com/kramirez/rex-stub/internal/simulation"
)

type Deps struct {
	APIKey  string
	State   *simulation.State
	Service *rex.Service
	Logger  *slog.Logger
	Random  func() float64
}

func NewHandler(deps Deps) http.Handler {
	business := http.NewServeMux()
	rex.NewEndpoint(deps.Service).Routes(business)

	admin := http.NewServeMux()
	simulation.Routes(admin, deps.State)

	root := http.NewServeMux()
	root.HandleFunc("GET /health", health)
	root.Handle("/api/v1/", middleware.Chain(business,
		middleware.APIKey(deps.APIKey),
		middleware.Simulation(deps.State, deps.Random),
	))
	root.Handle("/admin/", middleware.Chain(admin, middleware.APIKey(deps.APIKey)))
	root.HandleFunc("/", httpx.NotFound)

	return middleware.Logging(deps.Logger)(root)
}

func health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "UP"})
}
