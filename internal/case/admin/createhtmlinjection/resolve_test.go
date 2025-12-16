package createhtmlinjection_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/case/admin/createhtmlinjection"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createhtmlinjection_test . Env

type Env interface {
	InsertHTMLInjection(ctx context.Context, arg db.InsertHTMLInjectionParams) (db.HtmlInjection, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateHTMLInjectionInput
	}

	activeFrom := time.Now()
	activeTo := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name    string
		env     createhtmlinjection.Env
		args    args
		want    model.CreateHTMLInjectionOrErrorPayload
		wantErr bool
	}{
		{
			name: "successful create HTML injection",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				InsertHTMLInjectionFunc: func(ctx context.Context, arg db.InsertHTMLInjectionParams) (db.HtmlInjection, error) {
					return db.HtmlInjection{
						ID:          123,
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
				input: model.CreateHTMLInjectionInput{
					Description: "Test injection",
					Position:    0,
					Placement:   "head",
					Content:     "<script>console.log('test');</script>",
					ActiveFrom:  &activeFrom,
					ActiveTo:    &activeTo,
				},
			},
			want: &model.CreateHTMLInjectionPayload{
				HTMLInjection: &db.HtmlInjection{
					ID:          123,
					CreatedAt:   time.Time{}, // Will be set by InsertHTMLInjectionFunc
					ActiveFrom:  &activeFrom,
					ActiveTo:    &activeTo,
					Description: "Test injection",
					Position:    0,
					Placement:   "head",
					Content:     "<script>console.log('test');</script>",
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
				input: model.CreateHTMLInjectionInput{
					Description: "Test injection",
					Position:    0,
					Placement:   "head",
					Content:     "<script>console.log('test');</script>",
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
				input: model.CreateHTMLInjectionInput{
					Description: "",
					Position:    0,
					Placement:   "head",
					Content:     "<script>console.log('test');</script>",
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
			name: "validation error - invalid placement",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateHTMLInjectionInput{
					Description: "Test injection",
					Position:    0,
					Placement:   "invalid",
					Content:     "<script>console.log('test');</script>",
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "placement", Value: "must be a valid value"},
				},
			},
			wantErr: false,
		},
		{
			name: "database insert error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				InsertHTMLInjectionFunc: func(ctx context.Context, arg db.InsertHTMLInjectionParams) (db.HtmlInjection, error) {
					return db.HtmlInjection{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateHTMLInjectionInput{
					Description: "Test injection",
					Position:    0,
					Placement:   "head",
					Content:     "<script>console.log('test');</script>",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createhtmlinjection.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Ignore CreatedAt comparison for successful creates
			if payload, ok := got.(*model.CreateHTMLInjectionPayload); ok && payload != nil && payload.HTMLInjection != nil {
				payload.HTMLInjection.CreatedAt = time.Time{}
			}

			diff := pretty.Diff(got, tt.want)
			if len(diff) > 0 {
				t.Errorf("Resolve() diff:\n%s", pretty.Sprint(diff))
			}
		})
	}
}
