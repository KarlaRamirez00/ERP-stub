package rex

type ContractRequest struct {
	ExternalID   string  `json:"externalId"`
	StaffRut     string  `json:"staffRut"`
	ContractType string  `json:"contractType"`
	StartDate    string  `json:"startDate"`
	EndDate      string  `json:"endDate"`
	BaseSalary   float64 `json:"baseSalary"`
	ProjectCode  string  `json:"projectCode"`
}

type TerminationRequest struct {
	ExternalID              string `json:"externalId"`
	ContractReferenceNumber string `json:"contractReferenceNumber"`
	TerminationDate         string `json:"terminationDate"`
	LegalCause              string `json:"legalCause"`
	Definitive              bool   `json:"definitive"`
}

type AnnulmentRequest struct {
	Reason string `json:"reason"`
}

type AnnexRequest struct {
	ExternalID              string  `json:"externalId"`
	ContractReferenceNumber string  `json:"contractReferenceNumber"`
	AnnexType               string  `json:"annexType"`
	EffectiveDate           string  `json:"effectiveDate"`
	NewBaseSalary           float64 `json:"newBaseSalary"`
}

type ReferenceResponse struct {
	ReferenceNumber string `json:"referenceNumber"`
}
