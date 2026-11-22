package rex_test

import (
	"errors"
	"testing"

	"github.com/kramirez/rex-stub/internal/rex"
)

func newService() *rex.Service {
	return rex.NewService(rex.NewRepository())
}

func validContract(externalID string) rex.ContractRequest {
	return rex.ContractRequest{
		ExternalID:   externalID,
		StaffRut:     "12.345.678-9",
		ContractType: "O",
		StartDate:    "2026-12-01",
		EndDate:      "2027-03-01",
		BaseSalary:   850000,
		ProjectCode:  "PRJ-01",
	}
}

func syncedContract(t *testing.T, service *rex.Service) string {
	t.Helper()
	result, err := service.SyncContract(validContract("1"))
	if err != nil {
		t.Fatalf("error inesperado al sincronizar contrato: %v", err)
	}
	return result.ReferenceNumber
}

func TestSyncContractAssignsSequentialReferences(t *testing.T) {
	service := newService()

	first, _ := service.SyncContract(validContract("1"))
	second, _ := service.SyncContract(validContract("2"))

	if first.ReferenceNumber != "REX-000001" || second.ReferenceNumber != "REX-000002" {
		t.Fatalf("referencias inesperadas: %s, %s", first.ReferenceNumber, second.ReferenceNumber)
	}
	if !first.Created || !second.Created {
		t.Fatal("ambos contratos debían crearse")
	}
}

func TestSyncContractIsIdempotentByExternalID(t *testing.T) {
	service := newService()

	first, _ := service.SyncContract(validContract("1"))
	again, err := service.SyncContract(validContract("1"))

	if err != nil || again.Created || again.ReferenceNumber != first.ReferenceNumber {
		t.Fatalf("se esperaba la misma referencia sin crear: %+v, err=%v", again, err)
	}
}

func TestSyncContractReportsAllInvalidFields(t *testing.T) {
	_, err := newService().SyncContract(rex.ContractRequest{})

	var validation *rex.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("se esperaba ValidationError, fue %v", err)
	}
	for _, field := range []string{"externalId", "staffRut", "contractType", "startDate", "baseSalary", "projectCode"} {
		if _, found := validation.Fields[field]; !found {
			t.Errorf("faltó informar el campo %s", field)
		}
	}
	if _, found := validation.Fields["endDate"]; found {
		t.Error("endDate es opcional y no debía informarse")
	}
}

func TestSyncContractRejectsEndDateBeforeStartDate(t *testing.T) {
	req := validContract("1")
	req.EndDate = "2026-11-01"

	_, err := newService().SyncContract(req)

	var validation *rex.ValidationError
	if !errors.As(err, &validation) || validation.Fields["endDate"] == "" {
		t.Fatalf("se esperaba error en endDate, fue %v", err)
	}
}

func TestSyncContractRejectsMalformedDate(t *testing.T) {
	req := validContract("1")
	req.StartDate = "01-12-2026"

	_, err := newService().SyncContract(req)

	var validation *rex.ValidationError
	if !errors.As(err, &validation) || validation.Fields["startDate"] == "" {
		t.Fatalf("se esperaba error en startDate, fue %v", err)
	}
}

func TestSyncTerminationRequiresKnownContract(t *testing.T) {
	_, err := newService().SyncTermination(rex.TerminationRequest{
		ExternalID: "1", ContractReferenceNumber: "REX-000099", TerminationDate: "2027-01-15", LegalCause: "ART_159_4",
	})

	var unprocessable *rex.UnprocessableError
	if !errors.As(err, &unprocessable) {
		t.Fatalf("se esperaba UnprocessableError, fue %v", err)
	}
}

func TestSyncTerminationReturnsTerminationReference(t *testing.T) {
	service := newService()
	contract := syncedContract(t, service)

	result, err := service.SyncTermination(rex.TerminationRequest{
		ExternalID: "1", ContractReferenceNumber: contract, TerminationDate: "2027-01-15", LegalCause: "ART_159_4",
	})

	if err != nil || result.ReferenceNumber != "REX-TERM-000001" || !result.Created {
		t.Fatalf("resultado inesperado: %+v, err=%v", result, err)
	}
}

func TestAnnulTerminationIsIdempotent(t *testing.T) {
	service := newService()
	contract := syncedContract(t, service)
	termination, _ := service.SyncTermination(rex.TerminationRequest{
		ExternalID: "1", ContractReferenceNumber: contract, TerminationDate: "2027-01-15", LegalCause: "ART_159_4",
	})

	for range 2 {
		result, err := service.AnnulTermination(termination.ReferenceNumber, rex.AnnulmentRequest{Reason: "Error de digitación"})
		if err != nil || result.ReferenceNumber != termination.ReferenceNumber || result.Created {
			t.Fatalf("resultado inesperado: %+v, err=%v", result, err)
		}
	}
}

func TestAnnulTerminationRejectsUnknownOrWrongKindReference(t *testing.T) {
	service := newService()
	contract := syncedContract(t, service)

	for _, reference := range []string{"REX-TERM-000099", contract} {
		_, err := service.AnnulTermination(reference, rex.AnnulmentRequest{Reason: "motivo"})

		var notFound *rex.NotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("referencia %s: se esperaba NotFoundError, fue %v", reference, err)
		}
	}
}

func TestAnnulTerminationRequiresReason(t *testing.T) {
	_, err := newService().AnnulTermination("REX-TERM-000001", rex.AnnulmentRequest{})

	var validation *rex.ValidationError
	if !errors.As(err, &validation) || validation.Fields["reason"] == "" {
		t.Fatalf("se esperaba error en reason, fue %v", err)
	}
}

func TestSyncAnnexRequiresKnownContract(t *testing.T) {
	_, err := newService().SyncAnnex(rex.AnnexRequest{
		ExternalID: "1", ContractReferenceNumber: "REX-000099", AnnexType: "SALARY", EffectiveDate: "2027-02-01",
	})

	var unprocessable *rex.UnprocessableError
	if !errors.As(err, &unprocessable) {
		t.Fatalf("se esperaba UnprocessableError, fue %v", err)
	}
}

func TestSyncAnnexReturnsAnnexReference(t *testing.T) {
	service := newService()
	contract := syncedContract(t, service)

	result, err := service.SyncAnnex(rex.AnnexRequest{
		ExternalID: "1", ContractReferenceNumber: contract, AnnexType: "SALARY", EffectiveDate: "2027-02-01", NewBaseSalary: 900000,
	})

	if err != nil || result.ReferenceNumber != "REX-ANX-000001" {
		t.Fatalf("resultado inesperado: %+v, err=%v", result, err)
	}
}
