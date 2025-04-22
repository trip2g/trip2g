package model

func NewFieldError(field string, message string) *ErrorPayload {
	return &ErrorPayload{
		ByFields: []FieldMessage{{Name: field, Value: message}},
	}
}
