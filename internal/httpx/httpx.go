package httpx

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

const (
	CodeValidation    = "VALIDATION_ERROR"
	CodeUnauthorized  = "UNAUTHORIZED"
	CodeNotFound      = "NOT_FOUND"
	CodeUnprocessable = "UNPROCESSABLE"
	CodeUnavailable   = "REX_UNAVAILABLE"
	CodeInternal      = "INTERNAL_ERROR"

	maxBodyBytes = 1 << 20
)

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message, Fields: fields}})
}

func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteError(w, http.StatusBadRequest, CodeValidation, FieldsMessage(fields), fields)
}

func FieldsMessage(fields map[string]string) string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return "Datos inválidos: " + strings.Join(names, ", ")
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidation, "El cuerpo de la solicitud no es un JSON válido.", nil)
		return false
	}
	return true
}

func NotFound(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotFound, CodeNotFound, "Recurso no encontrado.", nil)
}
