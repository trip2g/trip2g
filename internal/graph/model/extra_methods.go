package model

import (
	"errors"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

func NewFieldError(field string, message string) *ErrorPayload {
	return &ErrorPayload{
		ByFields: []FieldMessage{{Name: field, Value: message}},
	}
}

func NewOzzoError(err error) *ErrorPayload {
	if err == nil {
		return nil
	}

	var ozzoErrors ozzo.Errors
	if !errors.As(err, &ozzoErrors) {
		return &ErrorPayload{Message: err.Error()}
	}

	payload := ErrorPayload{}

	for key, fieldErr := range ozzoErrors {
		payload.ByFields = append(payload.ByFields, FieldMessage{
			Name:  key,
			Value: fieldErr.Error(),
		})
	}

	return &payload
}
