package updatehtmlinjection_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/updatehtmlinjection"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updatehtmlinjection_test . Env

type Env interface {
	UpdateHTMLInjection(ctx context.Context, arg db.UpdateHTMLInjectionParams) (db.HtmlInjection, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.UpdateHTMLInjectionInput
	}

	activeFrom := time.Now()
	activeTo := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name    string
		env     updatehtmlinjection.Env
		args    args
		want    model.UpdateHTMLInjectionOrErrorPayload
		wantErr bool
	}{
		{
			name: "successful update HTML injection",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateHTMLInjectionFunc: func(ctx context.Context, arg db.UpdateHTMLInjectionParams) (db.HtmlInjection, error) {
					return db.HtmlInjection{
						ID:          arg.ID,
						CreatedAt:   time.Now(),
						ActiveFrom:  arg.ActiveFrom,
						ActiveTo:    arg.ActiveTo,
						Description: arg.Description,
						Position:    arg.Position,
						Placement:   arg.Placement,
						Content:     arg.Content,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateHTMLInjectionInput{
					ID:          123,
					Description: "Updated injection",
					Position:    1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
					ActiveFrom:  &activeFrom,
					ActiveTo:    &activeTo,
				},
			},
			want: &model.UpdateHTMLInjectionPayload{
				HTMLInjection: &db.HtmlInjection{
					ID:          123,
					CreatedAt:   time.Time{}, // Will be set by UpdateHTMLInjectionFunc
					ActiveFrom:  sql.NullTime{Time: activeFrom, Valid: true},
					ActiveTo:    sql.NullTime{Time: activeTo, Valid: true},
					Description: "Updated injection",
					Position:    1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
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
				input: model.UpdateHTMLInjectionInput{
					ID:          123,
					Description: "Updated injection",
					Position:    1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
				},
			},
			want:    nil,
			wantErr: true,
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
				input: model.UpdateHTMLInjectionInput{
					ID:          123,
					Description: "",
					Position:    1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
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
			name: "validation error - negative position",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateHTMLInjectionInput{
					ID:          123,
					Description: "Updated injection",
					Position:    -1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "position", Value: "must be no less than 0"},
				},
			},
			wantErr: false,
		},
		{
			name: "database update error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateHTMLInjectionFunc: func(ctx context.Context, arg db.UpdateHTMLInjectionParams) (db.HtmlInjection, error) {
					return db.HtmlInjection{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateHTMLInjectionInput{
					ID:          123,
					Description: "Updated injection",
					Position:    1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "update without dates",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateHTMLInjectionFunc: func(ctx context.Context, arg db.UpdateHTMLInjectionParams) (db.HtmlInjection, error) {
					return db.HtmlInjection{
						ID:          arg.ID,
						CreatedAt:   time.Now(),
						ActiveFrom:  arg.ActiveFrom,
						ActiveTo:    arg.ActiveTo,
						Description: arg.Description,
						Position:    arg.Position,
						Placement:   arg.Placement,
						Content:     arg.Content,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateHTMLInjectionInput{
					ID:          123,
					Description: "Updated injection",
					Position:    1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
					ActiveFrom:  nil,
					ActiveTo:    nil,
				},
			},
			want: &model.UpdateHTMLInjectionPayload{
				HTMLInjection: &db.HtmlInjection{
					ID:          123,
					CreatedAt:   time.Time{}, // Will be set by UpdateHTMLInjectionFunc
					ActiveFrom:  sql.NullTime{Valid: false},
					ActiveTo:    sql.NullTime{Valid: false},
					Description: "Updated injection",
					Position:    1,
					Placement:   "body_end",
					Content:     "<script>console.log('updated');</script>",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updatehtmlinjection.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Ignore CreatedAt comparison for successful updates
			if payload, ok := got.(*model.UpdateHTMLInjectionPayload); ok && payload != nil && payload.HTMLInjection != nil {
				payload.HTMLInjection.CreatedAt = time.Time{}
			}

			diff := pretty.Diff(got, tt.want)
			if len(diff) > 0 {
				t.Errorf("Resolve() diff:\n%s", pretty.Sprint(diff))
			}
		})
	}
}
