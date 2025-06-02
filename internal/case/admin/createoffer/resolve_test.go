package createoffer_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/createoffer"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createoffer_test . Env

type Env interface {
	InsertOffer(ctx context.Context, arg db.InsertOfferParams) (db.Offer, error)
	InsertOfferSubgraph(ctx context.Context, arg db.InsertOfferSubgraphParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	SubgraphByID(ctx context.Context, id int64) (db.Subgraph, error)
	GenerateUniqID() string
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateOfferInput
	}
	
	startsAt := time.Now()
	endsAt := time.Now().Add(24 * time.Hour)
	lifetime := "1 month"
	
	tests := []struct {
		name          string
		env           createoffer.Env
		args          args
		want          model.CreateOfferOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful create offer",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				GenerateUniqIDFunc: func() string {
					return "generated-offer-id"
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{
						ID:   id,
						Name: "test-subgraph",
					}, nil
				},
				InsertOfferFunc: func(ctx context.Context, arg db.InsertOfferParams) (db.Offer, error) {
					return db.Offer{
						ID:       123,
						PublicID: arg.PublicID,
						PriceUsd: arg.PriceUsd,
						Lifetime: arg.Lifetime,
						StartsAt: arg.StartsAt,
						EndsAt:   arg.EndsAt,
					}, nil
				},
				InsertOfferSubgraphFunc: func(ctx context.Context, arg db.InsertOfferSubgraphParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateOfferInput{
					PriceUsd:     9.99,
					Lifetime:     &lifetime,
					StartsAt:     &startsAt,
					EndsAt:       &endsAt,
					SubgraphIds:  []int64{1, 2},
				},
			},
			want: &model.CreateOfferPayload{
				Offer: &db.Offer{
					ID:       123,
					PublicID: "generated-offer-id",
					PriceUsd: sql.NullFloat64{Float64: 9.99, Valid: true},
					Lifetime: func() *appmodel.Lifetime { l := appmodel.Lifetime(lifetime); return &l }(),
					StartsAt: sql.NullTime{Time: startsAt, Valid: true},
					EndsAt:   sql.NullTime{Time: endsAt, Valid: true},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateUniqIDCalls()))
				require.Equal(t, 2, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 1, len(mockEnv.InsertOfferCalls()))
				require.Equal(t, 2, len(mockEnv.InsertOfferSubgraphCalls()))
				
				// Verify subgraph IDs were checked
				require.Equal(t, int64(1), mockEnv.SubgraphByIDCalls()[0].ID)
				require.Equal(t, int64(2), mockEnv.SubgraphByIDCalls()[1].ID)
				
				// Verify offer params
				offerParams := mockEnv.InsertOfferCalls()[0].Arg
				require.Equal(t, "generated-offer-id", offerParams.PublicID)
				require.Equal(t, 9.99, offerParams.PriceUsd.Float64)
				require.True(t, offerParams.PriceUsd.Valid)
				require.NotNil(t, offerParams.Lifetime)
				require.Equal(t, lifetime, string(*offerParams.Lifetime))
				
				// Verify offer-subgraph associations
				require.Equal(t, int64(123), mockEnv.InsertOfferSubgraphCalls()[0].Arg.OfferID)
				require.Equal(t, int64(1), mockEnv.InsertOfferSubgraphCalls()[0].Arg.SubgraphID)
				require.Equal(t, int64(123), mockEnv.InsertOfferSubgraphCalls()[1].Arg.OfferID)
				require.Equal(t, int64(2), mockEnv.InsertOfferSubgraphCalls()[1].Arg.SubgraphID)
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
				input: model.CreateOfferInput{
					PriceUsd:    9.99,
					SubgraphIds: []int64{1},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.GenerateUniqIDCalls()))
				require.Equal(t, 0, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferSubgraphCalls()))
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
				input: model.CreateOfferInput{
					PriceUsd:    -0.01,
					SubgraphIds: []int64{1},
				},
			},
			want:    &model.ErrorPayload{ByFields: []model.FieldMessage{{Name: "priceUSD", Value: "must be no less than 0"}}},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.GenerateUniqIDCalls()))
				require.Equal(t, 0, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferSubgraphCalls()))
			},
		},
		{
			name: "validation error - no subgraphs",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateOfferInput{
					PriceUsd:    9.99,
					SubgraphIds: []int64{},
				},
			},
			want:    &model.ErrorPayload{ByFields: []model.FieldMessage{{Name: "subgraphIds", Value: "cannot be blank"}}},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.GenerateUniqIDCalls()))
				require.Equal(t, 0, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferSubgraphCalls()))
			},
		},
		{
			name: "subgraph not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				GenerateUniqIDFunc: func() string {
					return "generated-offer-id"
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{}, errors.New("not found")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateOfferInput{
					PriceUsd:    9.99,
					SubgraphIds: []int64{999},
				},
			},
			want: &model.ErrorPayload{Message: "subgraph with ID 999 does not exist"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateUniqIDCalls()))
				require.Equal(t, 1, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferSubgraphCalls()))
				
				require.Equal(t, int64(999), mockEnv.SubgraphByIDCalls()[0].ID)
			},
		},
		{
			name: "insert offer error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				GenerateUniqIDFunc: func() string {
					return "generated-offer-id"
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{ID: id, Name: "test-subgraph"}, nil
				},
				InsertOfferFunc: func(ctx context.Context, arg db.InsertOfferParams) (db.Offer, error) {
					return db.Offer{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateOfferInput{
					PriceUsd:    9.99,
					SubgraphIds: []int64{1},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateUniqIDCalls()))
				require.Equal(t, 1, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 1, len(mockEnv.InsertOfferCalls()))
				require.Equal(t, 0, len(mockEnv.InsertOfferSubgraphCalls()))
			},
		},
		{
			name: "insert offer subgraph error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				GenerateUniqIDFunc: func() string {
					return "generated-offer-id"
				},
				SubgraphByIDFunc: func(ctx context.Context, id int64) (db.Subgraph, error) {
					return db.Subgraph{ID: id, Name: "test-subgraph"}, nil
				},
				InsertOfferFunc: func(ctx context.Context, arg db.InsertOfferParams) (db.Offer, error) {
					return db.Offer{
						ID:       123,
						PublicID: arg.PublicID,
						PriceUsd: arg.PriceUsd,
					}, nil
				},
				InsertOfferSubgraphFunc: func(ctx context.Context, arg db.InsertOfferSubgraphParams) error {
					return errors.New("foreign key constraint failed")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateOfferInput{
					PriceUsd:    9.99,
					SubgraphIds: []int64{1},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateUniqIDCalls()))
				require.Equal(t, 1, len(mockEnv.SubgraphByIDCalls()))
				require.Equal(t, 1, len(mockEnv.InsertOfferCalls()))
				require.Equal(t, 1, len(mockEnv.InsertOfferSubgraphCalls()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createoffer.Resolve(tt.args.ctx, tt.env, tt.args.input)
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