package submitform

import (
	"context"
	"fmt"
	"net/mail"

	"trip2g/internal/formspec"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg submitform_test . Env

type Env interface {
	GetFormSpec(ctx context.Context, noteVersionID int64, formID string) (*formspec.FormSpec, error)
	InsertFormSubmit(ctx context.Context, noteVersionID int64, formID string, userID *int64, ip string) (int64, error)
	InsertFormStringValue(ctx context.Context, submitID int64, fieldName, value string) error
	InsertFormIntValue(ctx context.Context, submitID int64, fieldName string, value int64) error
	InsertFormBoolValue(ctx context.Context, submitID int64, fieldName string, value bool) error
	EnqueueSendFormSubmitEmail(ctx context.Context, submitID int64) error
	RequestIP(ctx context.Context) string
	UserID(ctx context.Context) *int64
}

type FieldValue struct {
	Name        string
	StringValue *string
	IntValue    *int
	BoolValue   *bool
	FilePresent bool
}

type Input struct {
	NoteVersionID  int64
	FormID         string
	TurnstileToken string
	Fields         []FieldValue
}

type Payload interface{ isSubmitFormPayload() }

type SuccessResult struct{ SubmitID int64 }
type ErrorResult struct{ Message string }

func (s *SuccessResult) isSubmitFormPayload() {}
func (e *ErrorResult) isSubmitFormPayload()   {}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) { //nolint:gocognit
	spec, err := env.GetFormSpec(ctx, input.NoteVersionID, input.FormID)
	if err != nil {
		return nil, fmt.Errorf("submitform: get form spec: %w", err)
	}
	if spec == nil {
		return &ErrorResult{Message: "form_not_found"}, nil
	}

	if valErr := validateFields(spec, input.Fields); valErr != nil {
		return &ErrorResult{Message: valErr.Error()}, nil //nolint:nilerr // validation errors become ErrorResult, not Go errors
	}

	ip := env.RequestIP(ctx)
	userID := env.UserID(ctx)

	submitID, err := env.InsertFormSubmit(ctx, input.NoteVersionID, input.FormID, userID, ip)
	if err != nil {
		return nil, fmt.Errorf("submitform: insert submit: %w", err)
	}

	for _, fv := range input.Fields {
		field := findField(spec, fv.Name)
		if field == nil {
			continue
		}
		switch field.Type {
		case formspec.FieldTypeText, formspec.FieldTypeEmail:
			if fv.StringValue != nil {
				if insertErr := env.InsertFormStringValue(ctx, submitID, fv.Name, *fv.StringValue); insertErr != nil {
					return nil, fmt.Errorf("submitform: insert string %q: %w", fv.Name, insertErr)
				}
			}
		case formspec.FieldTypeInt:
			if fv.IntValue != nil {
				if insertErr := env.InsertFormIntValue(ctx, submitID, fv.Name, int64(*fv.IntValue)); insertErr != nil {
					return nil, fmt.Errorf("submitform: insert int %q: %w", fv.Name, insertErr)
				}
			}
		case formspec.FieldTypeBool:
			if fv.BoolValue != nil {
				if insertErr := env.InsertFormBoolValue(ctx, submitID, fv.Name, *fv.BoolValue); insertErr != nil {
					return nil, fmt.Errorf("submitform: insert bool %q: %w", fv.Name, insertErr)
				}
			}
		case formspec.FieldTypeFile:
			if fv.FilePresent {
				return &ErrorResult{Message: "file_upload_not_supported"}, nil
			}
		}
	}

	_ = env.EnqueueSendFormSubmitEmail(ctx, submitID) // non-fatal

	return &SuccessResult{SubmitID: submitID}, nil
}

func validateFields(spec *formspec.FormSpec, values []FieldValue) error { //nolint:gocognit,gocyclo,cyclop // field-by-field validation, complexity is inherent
	valueMap := make(map[string]FieldValue, len(values))
	for _, fv := range values {
		valueMap[fv.Name] = fv
	}
	for _, field := range spec.Fields {
		fv, present := valueMap[field.Name]
		if !present {
			if field.Required {
				return fmt.Errorf("%s: required", field.Name)
			}
			continue
		}
		switch field.Type {
		case formspec.FieldTypeText:
			if fv.StringValue == nil {
				if field.Required {
					return fmt.Errorf("%s: required", field.Name)
				}
				continue
			}
			v := *fv.StringValue
			if field.MinLength != nil && len(v) < *field.MinLength {
				return fmt.Errorf("%s: too short", field.Name)
			}
			if field.MaxLength != nil && len(v) > *field.MaxLength {
				return fmt.Errorf("%s: too long", field.Name)
			}
			if len(field.StringEnum) > 0 && !containsStr(field.StringEnum, v) {
				return fmt.Errorf("%s: invalid value", field.Name)
			}
		case formspec.FieldTypeEmail:
			if fv.StringValue == nil {
				if field.Required {
					return fmt.Errorf("%s: required", field.Name)
				}
				continue
			}
			if _, parseErr := mail.ParseAddress(*fv.StringValue); parseErr != nil {
				return fmt.Errorf("%s: invalid email", field.Name)
			}
		case formspec.FieldTypeInt:
			if fv.IntValue == nil {
				if field.Required {
					return fmt.Errorf("%s: required", field.Name)
				}
				continue
			}
			v := *fv.IntValue
			if field.Min != nil && v < *field.Min {
				return fmt.Errorf("%s: too small", field.Name)
			}
			if field.Max != nil && v > *field.Max {
				return fmt.Errorf("%s: too large", field.Name)
			}
			if len(field.IntEnum) > 0 && !containsInt(field.IntEnum, v) {
				return fmt.Errorf("%s: invalid value", field.Name)
			}
		case formspec.FieldTypeBool:
			if fv.BoolValue == nil {
				if field.Required {
					return fmt.Errorf("%s: required", field.Name)
				}
				continue
			}
			if len(field.BoolEnum) > 0 && !containsBool(field.BoolEnum, *fv.BoolValue) {
				return fmt.Errorf("%s: invalid value", field.Name)
			}
		case formspec.FieldTypeFile:
			// file uploads are not validated server-side
		}
	}
	return nil
}

func findField(spec *formspec.FormSpec, name string) *formspec.FormField {
	for i := range spec.Fields {
		if spec.Fields[i].Name == name {
			return &spec.Fields[i]
		}
	}
	return nil
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func containsInt(ns []int, n int) bool {
	for _, v := range ns {
		if v == n {
			return true
		}
	}
	return false
}

func containsBool(bs []bool, b bool) bool {
	for _, v := range bs {
		if v == b {
			return true
		}
	}
	return false
}
