package simulation

import (
	"net/http"

	"github.com/kramirez/rex-stub/internal/httpx"
)

func Routes(mux *http.ServeMux, state *State) {
	mux.HandleFunc("GET /admin/simulation", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, state.Get())
	})
	mux.HandleFunc("PUT /admin/simulation", func(w http.ResponseWriter, r *http.Request) {
		var settings Settings
		if !httpx.DecodeJSON(w, r, &settings) {
			return
		}
		if fields := state.Set(settings); fields != nil {
			httpx.WriteValidationError(w, fields)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, state.Get())
	})
	mux.HandleFunc("/", httpx.NotFound)
}
