package updateoffer_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/updateoffer"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updateoffer_test . Env

type Env interface {
	UpdateOffer(ctx context.Context, arg db.UpdateOfferParams) (db.Offer, error)
	DeleteOfferSubgraphs(ctx context.Context, offerID int64) error
	InsertOfferSubgraph(ctx context.Context, arg db.InsertOfferSubgraphParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	OfferByID(ctx context.Context, id int64) (db.Offer, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.UpdateOfferInput
	}
	
	startsAt := time.Now()
	endsAt := time.Now().Add(24 * time.Hour)
	lifetime := "1 month"
	priceUSD := 19.99
	
	tests := []struct {
		name          string
		env           updateoffer.Env
		args          args
		want          model.UpdateOfferOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful update offer all fields",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				OfferByIDFunc: func(ctx context.Context, id int64) (db.Offer, error) {
					return db.Offer{
						ID:       id,
						PriceUsd: sql.NullFloat64{Float64: 9.99, Valid: true},
					}, nil
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{
						ID:   id,
						Name: "test-subgraph",
					}, nil
				},
				UpdateOfferFunc: func(ctx context.Context, arg db.UpdateOfferParams) (db.Offer, error) {
					return db.Offer{
						ID:       arg.ID,
						PriceUsd: arg.PriceUsd,
						Lifetime: arg.Lifetime,
						StartsAt: arg.StartsAt,
						EndsAt:   arg.EndsAt,
					}, nil
				},
				DeleteOfferSubgraphsFunc: func(ctx context.Context, offerID int64) error {
					return nil
				},
				InsertOfferSubgraphFunc: func(ctx context.Context, arg db.InsertOfferSubgraphParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:          123,
					PriceUsd:    &priceUSD,
					Lifetime:    &lifetime,
					StartsAt:    &startsAt,
					EndsAt:      &endsAt,
					SubgraphIds: []int64{1, 2},
				},
			},
			want: &model.UpdateOfferPayload{
				Offer: &db.Offer{
					ID:       123,
					PriceUsd: sql.NullFloat64{Float64: priceUSD, Valid: true},
					Lifetime: func() *appmodel.Lifetime { l := appmodel.Lifetime(lifetime); return &l }(),
					StartsAt: sql.NullTime{Time: startsAt, Valid: true},
					EndsAt:   sql.NullTime{Time: endsAt, Valid: true},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 2, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateOfferCalls()))
				require.Equal(t, 1, len(mockEnv.DeleteOfferSubgraphsCalls()))
				require.Equal(t, 2, len(mockEnv.InsertOfferSubgraphCalls()))
				
				// Verify offer ID was checked
				require.Equal(t, int64(123), mockEnv.OfferByIDCalls()[0].ID)
				
				// Verify subgraph IDs were checked
				require.Equal(t, int64(1), mockEnv.SubgraphByIDCalls()[0].ID)
				require.Equal(t, int64(2), mockEnv.SubgraphByIDCalls()[1].ID)
				
				// Verify update params
				updateParams := mockEnv.UpdateOfferCalls()[0].Arg
				require.Equal(t, int64(123), updateParams.ID)
				require.Equal(t, priceUSD, updateParams.PriceUsd.Float64)
				require.True(t, updateParams.PriceUsd.Valid)
			},
		},
		{
			name: "successful update offer without subgraphs",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				OfferByIDFunc: func(ctx context.Context, id int64) (db.Offer, error) {
					return db.Offer{
						ID:       id,
						PriceUsd: sql.NullFloat64{Float64: 9.99, Valid: true},
					}, nil
				},
				UpdateOfferFunc: func(ctx context.Context, arg db.UpdateOfferParams) (db.Offer, error) {
					return db.Offer{
						ID:       arg.ID,
						PriceUsd: arg.PriceUsd,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:       123,
					PriceUsd: &priceUSD,
				},
			},
			want: &model.UpdateOfferPayload{
				Offer: &db.Offer{
					ID:       123,
					PriceUsd: sql.NullFloat64{Float64: priceUSD, Valid: true},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 0, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateOfferCalls()))
				require.Equal(t, 0, len(mockEnv.DeleteOfferSubgraphsCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferSubgraphCalls()))
			},
		},
		{
			name: "admin token error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:       123,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.OfferByIDCalls()))
			},
		},
		{
			name: "offer not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				OfferByIDFunc: func(ctx context.Context, id int64) (db.Offer, error) {
					return db.Offer{}, errors.New("not found")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:       999,
				},
			},
			want: &model.ErrorPayload{Message: "offer not found"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, int64(999), mockEnv.OfferByIDCalls()[0].ID)
			},
		},
		{
			name: "validation error - invalid price",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:       123,
					PriceUsd: func() *float64 { p := -0.01; return &p }(),
				},
			},
			want: &model.ErrorPayload{ByFields: []model.FieldMessage{{Name: "priceUSD", Value: "must be no less than 0"}}},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 0, len(mockEnv.UpdateOfferCalls()))
			},
		},
		{
			name: "validation error - empty subgraphs array",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:          123,
					SubgraphIds: []int64{},
				},
			},
			want: &model.ErrorPayload{ByFields: []model.FieldMessage{{Name: "subgraphIds", Value: "cannot be blank"}}},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 0, len(mockEnv.SubgraphByIDCalls()))
			},
		},
		{
			name: "validation error - starts at after ends at",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:       123,
					StartsAt: func() *time.Time { t := time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC); return &t }(),
					EndsAt:   func() *time.Time { t := time.Date(2025, 6, 4, 0, 0, 0, 0, time.UTC); return &t }(),
				},
			},
			want: &model.ErrorPayload{ByFields: []model.FieldMessage{{Name: "startsAt", Value: "must be before ends at"}}},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 0, len(mockEnv.SubgraphByIDCalls()))
			},
		},
		{
			name: "subgraph not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				OfferByIDFunc: func(ctx context.Context, id int64) (db.Offer, error) {
					return db.Offer{ID: id}, nil
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{}, errors.New("not found")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:          123,
					SubgraphIds: []int64{999},
				},
			},
			want: &model.ErrorPayload{Message: "subgraph with ID 999 does not exist"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 1, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, int64(999), mockEnv.SubgraphByIDCalls()[0].ID)
			},
		},
		{
			name: "update offer error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				OfferByIDFunc: func(ctx context.Context, id int64) (db.Offer, error) {
					return db.Offer{ID: id}, nil
				},
				UpdateOfferFunc: func(ctx context.Context, arg db.UpdateOfferParams) (db.Offer, error) {
					return db.Offer{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:       123,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateOfferCalls()))
			},
		},
		{
			name: "delete offer subgraphs error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				OfferByIDFunc: func(ctx context.Context, id int64) (db.Offer, error) {
					return db.Offer{ID: id}, nil
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{ID: id}, nil
				},
				UpdateOfferFunc: func(ctx context.Context, arg db.UpdateOfferParams) (db.Offer, error) {
					return db.Offer{ID: arg.ID}, nil
				},
				DeleteOfferSubgraphsFunc: func(ctx context.Context, offerID int64) error {
					return errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateOfferInput{
					ID:          123,
					SubgraphIds: []int64{1},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.OfferByIDCalls()))
				require.Equal(t, 1, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateOfferCalls()))
				require.Equal(t, 1, len(mockEnv.DeleteOfferSubgraphsCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferSubgraphCalls()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateoffer.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Resolve() = %v, want %v", got, tt.want)
					for _, desc := range pretty.Diff(got, tt.want) {
						t.Error(desc)
					}
				}
			}

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}