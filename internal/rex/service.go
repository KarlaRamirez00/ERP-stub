package rex

import (
	"strings"
	"time"
)

const (
	dateLayout        = "2006-01-02"
	msgRequired       = "campo obligatorio"
	msgInvalidDate    = "debe tener formato AAAA-MM-DD"
	msgPositiveNumber = "debe ser mayor a 0"
)

type Result struct {
	ReferenceNumber string
	Created         bool
}

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "datos inválidos" }

type NotFoundError struct{ Message string }

func (e *NotFoundError) Error() string { return e.Message }

type UnprocessableError struct{ Message string }

func (e *UnprocessableError) Error() string { return e.Message }

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) SyncContract(req ContractRequest) (Result, error) {
	fields := fieldErrors{}
	fields.require("externalId", req.ExternalID)
	fields.require("staffRut", req.StaffRut)
	fields.require("contractType", req.ContractType)
	fields.require("projectCode", req.ProjectCode)
	start, startValid := fields.requireDate("startDate", req.StartDate)
	if strings.TrimSpace(req.EndDate) != "" {
		end, endValid := fields.requireDate("endDate", req.EndDate)
		if startValid && endValid && end.Before(start) {
			fields["endDate"] = "no puede ser anterior a startDate"
		}
	}
	if req.BaseSalary <= 0 {
		fields["baseSalary"] = msgPositiveNumber
	}
	if err := fields.asError(); err != nil {
		return Result{}, err
	}
	return s.register(KindContract, req.ExternalID), nil
}

func (s *Service) SyncTermination(req TerminationRequest) (Result, error) {
	fields := fieldErrors{}
	fields.require("externalId", req.ExternalID)
	fields.require("contractReferenceNumber", req.ContractReferenceNumber)
	fields.require("legalCause", req.LegalCause)
	fields.requireDate("terminationDate", req.TerminationDate)
	if err := fields.asError(); err != nil {
		return Result{}, err
	}
	if err := s.requireContract(req.ContractReferenceNumber); err != nil {
		return Result{}, err
	}
	return s.register(KindTermination, req.ExternalID), nil
}

func (s *Service) AnnulTermination(reference string, req AnnulmentRequest) (Result, error) {
	fields := fieldErrors{}
	fields.require("reason", req.Reason)
	if err := fields.asError(); err != nil {
		return Result{}, err
	}
	if !s.repository.Exists(KindTermination, reference) {
		return Result{}, &NotFoundError{Message: "No existe una terminación con la referencia " + reference + "."}
	}
	return Result{ReferenceNumber: reference, Created: false}, nil
}

func (s *Service) SyncAnnex(req AnnexRequest) (Result, error) {
	fields := fieldErrors{}
	fields.require("externalId", req.ExternalID)
	fields.require("contractReferenceNumber", req.ContractReferenceNumber)
	fields.require("annexType", req.AnnexType)
	fields.requireDate("effectiveDate", req.EffectiveDate)
	if req.NewBaseSalary < 0 {
		fields["newBaseSalary"] = "no puede ser negativo"
	}
	if err := fields.asError(); err != nil {
		return Result{}, err
	}
	if err := s.requireContract(req.ContractReferenceNumber); err != nil {
		return Result{}, err
	}
	return s.register(KindAnnex, req.ExternalID), nil
}

func (s *Service) register(kind Kind, externalID string) Result {
	reference, created := s.repository.Register(kind, strings.TrimSpace(externalID))
	return Result{ReferenceNumber: reference, Created: created}
}

func (s *Service) requireContract(reference string) error {
	if s.repository.Exists(KindContract, strings.TrimSpace(reference)) {
		return nil
	}
	return &UnprocessableError{Message: "El contrato " + reference + " no existe en Rex+."}
}

type fieldErrors map[string]string

func (f fieldErrors) require(name, value string) {
	if strings.TrimSpace(value) == "" {
		f[name] = msgRequired
	}
}

func (f fieldErrors) requireDate(name, value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		f[name] = msgRequired
		return time.Time{}, false
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		f[name] = msgInvalidDate
		return time.Time{}, false
	}
	return parsed, true
}

func (f fieldErrors) asError() error {
	if len(f) == 0 {
		return nil
	}
	return &ValidationError{Fields: f}
}
