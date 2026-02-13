package deletefrontmatterpatch_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/deletefrontmatterpatch"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg deletefrontmatterpatch_test . Env

type Env interface {
	DeleteFrontmatterPatch(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.DeleteFrontmatterPatchInput
	}

	tests := []struct {
		name    string
		env     deletefrontmatterpatch.Env
		args    args
		want    model.DeleteFrontmatterPatchOrErrorPayload
		wantErr bool
	}{
		{
			name: "successful delete frontmatter patch",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				DeleteFrontmatterPatchFunc: func(ctx context.Context, id int64) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteFrontmatterPatchInput{
					ID: 123,
				},
			},
			want: &model.DeleteFrontmatterPatchPayload{
				DeletedID: 123,
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
				input: model.DeleteFrontmatterPatchInput{
					ID: 123,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "validation error - zero ID",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteFrontmatterPatchInput{
					ID: 0,
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "id", Value: "cannot be blank"},
				},
			},
			wantErr: false,
		},
		{
			name: "database delete error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				DeleteFrontmatterPatchFunc: func(ctx context.Context, id int64) error {
					return errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteFrontmatterPatchInput{
					ID: 123,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not found error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				DeleteFrontmatterPatchFunc: func(ctx context.Context, id int64) error {
					return errors.New("no rows affected")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.DeleteFrontmatterPatchInput{
					ID: 999,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deletefrontmatterpatch.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			diff := pretty.Diff(got, tt.want)
			if len(diff) > 0 {
				t.Errorf("Resolve() diff:\n%s", pretty.Sprint(diff))
			}
		})
	}
}
