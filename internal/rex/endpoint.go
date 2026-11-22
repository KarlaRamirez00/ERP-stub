package rex

import (
	"errors"
	"net/http"

	"github.com/kramirez/rex-stub/internal/httpx"
)

type Endpoint struct {
	service *Service
}

func NewEndpoint(service *Service) *Endpoint {
	return &Endpoint{service: service}
}

func (e *Endpoint) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/contracts", e.syncContract)
	mux.HandleFunc("POST /api/v1/terminations", e.syncTermination)
	mux.HandleFunc("POST /api/v1/terminations/{reference}/annulment", e.annulTermination)
	mux.HandleFunc("POST /api/v1/annexes", e.syncAnnex)
	mux.HandleFunc("/", httpx.NotFound)
}

func (e *Endpoint) syncContract(w http.ResponseWriter, r *http.Request) {
	var req ContractRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	result, err := e.service.SyncContract(req)
	respond(w, result, err)
}

func (e *Endpoint) syncTermination(w http.ResponseWriter, r *http.Request) {
	var req TerminationRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	result, err := e.service.SyncTermination(req)
	respond(w, result, err)
}

func (e *Endpoint) annulTermination(w http.ResponseWriter, r *http.Request) {
	var req AnnulmentRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	result, err := e.service.AnnulTermination(r.PathValue("reference"), req)
	respond(w, result, err)
}

func (e *Endpoint) syncAnnex(w http.ResponseWriter, r *http.Request) {
	var req AnnexRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	result, err := e.service.SyncAnnex(req)
	respond(w, result, err)
}

func respond(w http.ResponseWriter, result Result, err error) {
	if err != nil {
		writeServiceError(w, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, ReferenceResponse{ReferenceNumber: result.ReferenceNumber})
}

func writeServiceError(w http.ResponseWriter, err error) {
	var validation *ValidationError
	var notFound *NotFoundError
	var unprocessable *UnprocessableError

	switch {
	case errors.As(err, &validation):
		httpx.WriteValidationError(w, validation.Fields)
	case errors.As(err, &notFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, notFound.Message, nil)
	case errors.As(err, &unprocessable):
		httpx.WriteError(w, http.StatusUnprocessableEntity, httpx.CodeUnprocessable, unprocessable.Message, nil)
	default:
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "Error interno del servicio.", nil)
	}
}
