package createfrontmatterpatch_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/createfrontmatterpatch"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createfrontmatterpatch_test . Env

type Env interface {
	InsertFrontmatterPatch(ctx context.Context, arg db.InsertFrontmatterPatchParams) (db.NoteFrontmatterPatch, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateFrontmatterPatchInput
	}

	tests := []struct {
		name    string
		env     createfrontmatterpatch.Env
		args    args
		want    model.CreateFrontmatterPatchOrErrorPayload
		wantErr bool
	}{
		{
			name: "successful create frontmatter patch",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				InsertFrontmatterPatchFunc: func(ctx context.Context, arg db.InsertFrontmatterPatchParams) (db.NoteFrontmatterPatch, error) {
					return db.NoteFrontmatterPatch{
						ID:              123,
						IncludePatterns: arg.IncludePatterns,
						ExcludePatterns: arg.ExcludePatterns,
						Jsonnet:         arg.Jsonnet,
						Priority:        arg.Priority,
						Description:     arg.Description,
						Enabled:         arg.Enabled,
						CreatedBy:       arg.CreatedBy,
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{"docs/**"},
					ExcludePatterns: []string{"docs/private/**"},
					Jsonnet:         `{ draft: true }`,
					Priority:        10,
					Description:     "Add draft flag",
					Enabled:         true,
				},
			},
			want: &model.CreateFrontmatterPatchPayload{
				FrontmatterPatch: &db.NoteFrontmatterPatch{
					ID:              123,
					IncludePatterns: `["docs/**"]`,
					ExcludePatterns: `["docs/private/**"]`,
					Jsonnet:         `{ draft: true }`,
					Priority:        10,
					Description:     "Add draft flag",
					Enabled:         true,
					CreatedBy:       1,
					CreatedAt:       time.Time{},
					UpdatedAt:       time.Time{},
				},
			},
			wantErr: false,
		},
		{
			name: "successful create without exclude patterns",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				InsertFrontmatterPatchFunc: func(ctx context.Context, arg db.InsertFrontmatterPatchParams) (db.NoteFrontmatterPatch, error) {
					return db.NoteFrontmatterPatch{
						ID:              124,
						IncludePatterns: arg.IncludePatterns,
						ExcludePatterns: arg.ExcludePatterns,
						Jsonnet:         arg.Jsonnet,
						Priority:        arg.Priority,
						Description:     arg.Description,
						Enabled:         arg.Enabled,
						CreatedBy:       arg.CreatedBy,
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{"**/*.md"},
					Jsonnet:         `meta + { draft: true }`,
					Priority:        5,
					Description:     "Add draft to all",
					Enabled:         false,
				},
			},
			want: &model.CreateFrontmatterPatchPayload{
				FrontmatterPatch: &db.NoteFrontmatterPatch{
					ID:              124,
					IncludePatterns: `["**/*.md"]`,
					ExcludePatterns: `null`,
					Jsonnet:         `meta + { draft: true }`,
					Priority:        5,
					Description:     "Add draft to all",
					Enabled:         false,
					CreatedBy:       1,
					CreatedAt:       time.Time{},
					UpdatedAt:       time.Time{},
				},
			},
			wantErr: false,
		},
		{
			name: "failed admin token",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{"docs/**"},
					Jsonnet:         `meta + { draft: true }`,
					Priority:        10,
					Description:     "Add draft flag",
					Enabled:         true,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "validation error - empty include patterns",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{},
					Jsonnet:         `meta + { draft: true }`,
					Priority:        10,
					Description:     "Add draft flag",
					Enabled:         true,
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "includePatterns", Value: "cannot be blank"},
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - empty description",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{"docs/**"},
					Jsonnet:         `meta + { draft: true }`,
					Priority:        10,
					Description:     "",
					Enabled:         true,
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "description", Value: "cannot be blank"},
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - invalid glob pattern",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{"[invalid"},
					Jsonnet:         `meta + { draft: true }`,
					Priority:        10,
					Description:     "Test",
					Enabled:         true,
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "includePatterns", Value: "invalid glob pattern"},
				},
			},
			wantErr: false,
		},
		{
			name: "validation error - invalid jsonnet syntax",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{"docs/**"},
					Jsonnet:         `{ foo: }`,
					Priority:        10,
					Description:     "Test",
					Enabled:         true,
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "database insert error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				InsertFrontmatterPatchFunc: func(ctx context.Context, arg db.InsertFrontmatterPatchParams) (db.NoteFrontmatterPatch, error) {
					return db.NoteFrontmatterPatch{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateFrontmatterPatchInput{
					IncludePatterns: []string{"docs/**"},
					Jsonnet:         `meta + { draft: true }`,
					Priority:        10,
					Description:     "Add draft flag",
					Enabled:         true,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createfrontmatterpatch.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Ignore timestamp comparison for successful creates.
			if payload, ok := got.(*model.CreateFrontmatterPatchPayload); ok && payload != nil && payload.FrontmatterPatch != nil {
				payload.FrontmatterPatch.CreatedAt = time.Time{}
				payload.FrontmatterPatch.UpdatedAt = time.Time{}
			}

			// For invalid jsonnet, just check we got an ErrorPayload (message varies).
			if tt.name == "validation error - invalid jsonnet syntax" {
				if _, ok := got.(*model.ErrorPayload); !ok {
					t.Errorf("Resolve() expected *model.ErrorPayload for invalid jsonnet, got %T", got)
				}
				return
			}

			diff := pretty.Diff(got, tt.want)
			if len(diff) > 0 {
				t.Errorf("Resolve() diff:\n%s", pretty.Sprint(diff))
			}
		})
	}
}
