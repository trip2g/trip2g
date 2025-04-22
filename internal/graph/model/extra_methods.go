package model

import (
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

	errors, ok := err.(ozzo.Errors)
	if !ok {
		return &ErrorPayload{Message: err.Error()}
	}

	payload := ErrorPayload{}

	for key, fieldErr := range errors {
		payload.ByFields = append(payload.ByFields, FieldMessage{
			Name:  key,
			Value: fieldErr.Error(),
		})
	}

	return &payload
}
