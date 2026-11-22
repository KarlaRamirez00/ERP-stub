package server_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/kramirez/rex-stub/internal/httpx"
	"github.com/kramirez/rex-stub/internal/rex"
	"github.com/kramirez/rex-stub/internal/server"
	"github.com/kramirez/rex-stub/internal/simulation"
)

const (
	apiKey = "test-key"

	validContractBody = `{"externalId":"123","staffRut":"12.345.678-9","contractType":"O",
		"startDate":"2026-12-01","endDate":"2027-03-01","baseSalary":850000,"projectCode":"PRJ-01"}`
)

func newHandler(settings simulation.Settings) http.Handler {
	return server.NewHandler(server.Deps{
		APIKey:  apiKey,
		State:   simulation.NewState(settings),
		Service: rex.NewService(rex.NewRepository()),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Random:  func() float64 { return 0.99 },
	})
}

func do(handler http.Handler, method, path, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func reference(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body rex.ReferenceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v (%s)", err, rec.Body.String())
	}
	return body.ReferenceNumber
}

func errorBody(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorDetail {
	t.Helper()
	var body httpx.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta de error no es JSON válido: %v (%s)", err, rec.Body.String())
	}
	return body.Error
}

func TestCA1CreateContractReturnsReference(t *testing.T) {
	rec := do(newHandler(simulation.Settings{}), http.MethodPost, "/api/v1/contracts", validContractBody, apiKey)

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, fue %d (%s)", rec.Code, rec.Body.String())
	}
	if !regexp.MustCompile(`^REX-\d{6}$`).MatchString(reference(t, rec)) {
		t.Fatalf("formato de referencia inesperado: %s", rec.Body.String())
	}
}

func TestCA2RejectsMissingOrWrongAPIKey(t *testing.T) {
	handler := newHandler(simulation.Settings{})

	for _, key := range []string{"", "incorrecta"} {
		rec := do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, key)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("clave %q: esperado 401, fue %d", key, rec.Code)
		}
		if errorBody(t, rec).Code != httpx.CodeUnauthorized {
			t.Fatalf("clave %q: código de error inesperado: %s", key, rec.Body.String())
		}
	}
}

func TestCA3RepeatedExternalIDReturnsSameReference(t *testing.T) {
	handler := newHandler(simulation.Settings{})

	first := do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, apiKey)
	second := do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, apiKey)

	if first.Code != http.StatusCreated || second.Code != http.StatusOK {
		t.Fatalf("esperado 201 y luego 200, fue %d y %d", first.Code, second.Code)
	}
	if reference(t, first) != reference(t, second) {
		t.Fatal("la referencia debía repetirse")
	}
}

func TestCA4TerminationWithUnknownContractReturns422(t *testing.T) {
	body := `{"externalId":"1","contractReferenceNumber":"REX-000099","terminationDate":"2027-01-15","legalCause":"ART_159_4"}`

	rec := do(newHandler(simulation.Settings{}), http.MethodPost, "/api/v1/terminations", body, apiKey)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("esperado 422, fue %d", rec.Code)
	}
}

func TestCA5AnnulmentUnknownReturns404AndRepeatedReturns200(t *testing.T) {
	handler := newHandler(simulation.Settings{})
	annulBody := `{"reason":"Error de digitación"}`

	unknown := do(handler, http.MethodPost, "/api/v1/terminations/REX-TERM-000099/annulment", annulBody, apiKey)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, fue %d", unknown.Code)
	}

	contract := reference(t, do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, apiKey))
	terminationBody := `{"externalId":"1","contractReferenceNumber":"` + contract + `","terminationDate":"2027-01-15","legalCause":"ART_159_4"}`
	termination := reference(t, do(handler, http.MethodPost, "/api/v1/terminations", terminationBody, apiKey))

	for i := 0; i < 2; i++ {
		rec := do(handler, http.MethodPost, "/api/v1/terminations/"+termination+"/annulment", annulBody, apiKey)
		if rec.Code != http.StatusOK || reference(t, rec) != termination {
			t.Fatalf("anulación %d: esperado 200 con %s, fue %d (%s)", i+1, termination, rec.Code, rec.Body.String())
		}
	}
}

func TestCA6ForceErrorReturns503ButHealthStaysUp(t *testing.T) {
	handler := newHandler(simulation.Settings{ForceError: true})

	business := do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, apiKey)
	health := do(handler, http.MethodGet, "/health", "", "")

	if business.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperado 503, fue %d", business.Code)
	}
	if health.Code != http.StatusOK {
		t.Fatalf("/health debía seguir en 200, fue %d", health.Code)
	}
}

func TestCA7AdminChangesSimulationWithoutRestart(t *testing.T) {
	handler := newHandler(simulation.Settings{})

	on := do(handler, http.MethodPut, "/admin/simulation", `{"forceError":true}`, apiKey)
	failing := do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, apiKey)
	off := do(handler, http.MethodPut, "/admin/simulation", `{"forceError":false}`, apiKey)
	recovered := do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, apiKey)

	if on.Code != http.StatusOK || off.Code != http.StatusOK {
		t.Fatalf("PUT /admin/simulation debía responder 200, fue %d y %d", on.Code, off.Code)
	}
	if failing.Code != http.StatusServiceUnavailable {
		t.Fatalf("con forceError esperado 503, fue %d", failing.Code)
	}
	if recovered.Code != http.StatusCreated {
		t.Fatalf("sin forceError esperado 201, fue %d", recovered.Code)
	}
}

func TestAdminRequiresAPIKeyAndValidatesSettings(t *testing.T) {
	handler := newHandler(simulation.Settings{})

	if rec := do(handler, http.MethodGet, "/admin/simulation", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sin clave esperado 401, fue %d", rec.Code)
	}
	if rec := do(handler, http.MethodPut, "/admin/simulation", `{"failureRate":3}`, apiKey); rec.Code != http.StatusBadRequest {
		t.Fatalf("tasa inválida esperado 400, fue %d", rec.Code)
	}
}

func TestCA8MissingFieldsListsEveryOne(t *testing.T) {
	rec := do(newHandler(simulation.Settings{}), http.MethodPost, "/api/v1/contracts", `{}`, apiKey)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, fue %d", rec.Code)
	}
	detail := errorBody(t, rec)
	if len(detail.Fields) != 6 {
		t.Fatalf("se esperaban 6 campos inválidos, hubo %v", detail.Fields)
	}
	if !strings.Contains(detail.Message, "staffRut") {
		t.Fatalf("el mensaje debía nombrar los campos: %s", detail.Message)
	}
}

func TestMalformedJSONReturns400(t *testing.T) {
	rec := do(newHandler(simulation.Settings{}), http.MethodPost, "/api/v1/contracts", `{no es json`, apiKey)

	if rec.Code != http.StatusBadRequest || errorBody(t, rec).Code != httpx.CodeValidation {
		t.Fatalf("esperado 400 VALIDATION_ERROR, fue %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestUnknownPathReturnsJSON404(t *testing.T) {
	rec := do(newHandler(simulation.Settings{}), http.MethodGet, "/nada", "", "")

	if rec.Code != http.StatusNotFound || errorBody(t, rec).Code != httpx.CodeNotFound {
		t.Fatalf("esperado 404 NOT_FOUND, fue %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestAnnexFlowAfterContract(t *testing.T) {
	handler := newHandler(simulation.Settings{})
	contract := reference(t, do(handler, http.MethodPost, "/api/v1/contracts", validContractBody, apiKey))
	body := `{"externalId":"7","contractReferenceNumber":"` + contract + `","annexType":"SALARY","effectiveDate":"2027-02-01","newBaseSalary":900000}`

	rec := do(handler, http.MethodPost, "/api/v1/annexes", body, apiKey)

	if rec.Code != http.StatusCreated || reference(t, rec) != "REX-ANX-000001" {
		t.Fatalf("esperado 201 con REX-ANX-000001, fue %d (%s)", rec.Code, rec.Body.String())
	}
}
