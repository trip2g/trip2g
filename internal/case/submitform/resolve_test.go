package submitform_test

import (
	"context"
	"testing"

	"trip2g/internal/case/submitform"
	"trip2g/internal/formspec"

	"github.com/stretchr/testify/require"
)

func TestResolve_form_not_found(t *testing.T) {
	env := &EnvMock{
		GetFormSpecFunc: func(ctx context.Context, noteVersionID int64, formID string) (*formspec.FormSpec, error) {
			return nil, nil
		},
		RequestIPFunc: func(ctx context.Context) string { return "1.2.3.4" },
		UserIDFunc:    func(ctx context.Context) *int64 { return nil },
	}
	payload, err := submitform.Resolve(context.Background(), env, submitform.Input{NoteVersionID: 1})
	require.NoError(t, err)
	errP, ok := payload.(*submitform.ErrorResult)
	require.True(t, ok)
	require.Equal(t, "form_not_found", errP.Message)
}

func TestResolve_required_field_missing(t *testing.T) {
	spec := &formspec.FormSpec{
		CanSubmit: formspec.CanSubmitGuest,
		Fields:    []formspec.FormField{{Name: "email", Type: formspec.FieldTypeEmail, Required: true}},
	}
	env := &EnvMock{
		GetFormSpecFunc: func(ctx context.Context, noteVersionID int64, formID string) (*formspec.FormSpec, error) {
			return spec, nil
		},
		RequestIPFunc: func(ctx context.Context) string { return "" },
		UserIDFunc:    func(ctx context.Context) *int64 { return nil },
	}
	payload, err := submitform.Resolve(context.Background(), env, submitform.Input{
		NoteVersionID: 1,
		Fields:        []submitform.FieldValue{{Name: "email", StringValue: nil}},
	})
	require.NoError(t, err)
	errP, ok := payload.(*submitform.ErrorResult)
	require.True(t, ok)
	require.Contains(t, errP.Message, "email")
}

func TestResolve_file_type_not_supported(t *testing.T) {
	spec := &formspec.FormSpec{
		CanSubmit: formspec.CanSubmitGuest,
		Fields:    []formspec.FormField{{Name: "attach", Type: formspec.FieldTypeFile}},
	}
	env := &EnvMock{
		GetFormSpecFunc: func(ctx context.Context, noteVersionID int64, formID string) (*formspec.FormSpec, error) {
			return spec, nil
		},
		InsertFormSubmitFunc: func(ctx context.Context, noteVersionID int64, formID string, userID *int64, ip string) (int64, error) {
			return 1, nil
		},
		RequestIPFunc:                  func(ctx context.Context) string { return "" },
		UserIDFunc:                     func(ctx context.Context) *int64 { return nil },
		EnqueueSendFormSubmitEmailFunc: func(ctx context.Context, submitID int64) error { return nil },
	}
	payload, err := submitform.Resolve(context.Background(), env, submitform.Input{
		NoteVersionID: 1,
		Fields:        []submitform.FieldValue{{Name: "attach", FilePresent: true}},
	})
	require.NoError(t, err)
	errP, ok := payload.(*submitform.ErrorResult)
	require.True(t, ok)
	require.Equal(t, "file_upload_not_supported", errP.Message)
}

func TestResolve_success(t *testing.T) {
	spec := &formspec.FormSpec{
		CanSubmit: formspec.CanSubmitGuest,
		Fields: []formspec.FormField{
			{Name: "name", Type: formspec.FieldTypeText, Required: true},
			{Name: "score", Type: formspec.FieldTypeInt},
			{Name: "agree", Type: formspec.FieldTypeBool},
		},
	}
	var gotVersionID int64
	var gotStrings []string
	var emailEnqueued bool

	env := &EnvMock{
		GetFormSpecFunc: func(ctx context.Context, noteVersionID int64, formID string) (*formspec.FormSpec, error) {
			return spec, nil
		},
		InsertFormSubmitFunc: func(ctx context.Context, noteVersionID int64, formID string, userID *int64, ip string) (int64, error) {
			gotVersionID = noteVersionID
			return 99, nil
		},
		InsertFormStringValueFunc: func(ctx context.Context, submitID int64, fieldName, value string) error {
			gotStrings = append(gotStrings, fieldName+":"+value)
			return nil
		},
		InsertFormIntValueFunc: func(ctx context.Context, submitID int64, fieldName string, value int64) error {
			return nil
		},
		InsertFormBoolValueFunc: func(ctx context.Context, submitID int64, fieldName string, value bool) error {
			return nil
		},
		EnqueueSendFormSubmitEmailFunc: func(ctx context.Context, submitID int64) error {
			emailEnqueued = true
			return nil
		},
		RequestIPFunc: func(ctx context.Context) string { return "1.2.3.4" },
		UserIDFunc:    func(ctx context.Context) *int64 { return nil },
	}

	nameVal := "Alice"
	scoreVal := 5
	agreeVal := true
	payload, err := submitform.Resolve(context.Background(), env, submitform.Input{
		NoteVersionID: 7,
		Fields: []submitform.FieldValue{
			{Name: "name", StringValue: &nameVal},
			{Name: "score", IntValue: &scoreVal},
			{Name: "agree", BoolValue: &agreeVal},
		},
	})
	require.NoError(t, err)
	result, ok := payload.(*submitform.SuccessResult)
	require.True(t, ok, "expected SuccessResult got %T", payload)
	require.Equal(t, int64(99), result.SubmitID)
	require.Equal(t, int64(7), gotVersionID)
	require.Contains(t, gotStrings, "name:Alice")
	require.True(t, emailEnqueued)
}
