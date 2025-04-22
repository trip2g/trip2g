package caseerr

type FieldError struct {
	Name    string
	Message string
}

type CaseError struct {
	Message  string
	ByFields []FieldError
}

func (CaseError) IsRequestEmailSignInCodeOrErrorPayload() {}

func (CaseError) IsSignInOrErrorPayload() {}
