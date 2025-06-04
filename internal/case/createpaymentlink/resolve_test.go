package createpaymentlink_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/createpaymentlink"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/nowpayments"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createpaymentlink_test . Env

type Env interface {
	PublicURL() string
	CreateNowpaymentsInvoice(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error)
	ActiveOfferByPublicID(ctx context.Context, id string) (db.Offer, error)
	InsertPurchase(ctx context.Context, arg db.InsertPurchaseParams) error
	GeneratePurchaseID() string
	GenerateHotAuthToken(ctx context.Context, data appmodel.HotAuthToken) (string, error)
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
	UserByEmail(ctx context.Context, email string) (db.User, error)
	StorePurchaseToken(ctx context.Context, data appmodel.PurchaseToken) (string, error)
	RequestEmailSignInCode(ctx context.Context, email string) error
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx context.Context
		req model.CreatePaymentLinkInput
	}

	tests := []struct {
		name          string
		env           createpaymentlink.Env
		args          args
		want          model.CreatePaymentLinkOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful payment link creation - authenticated user",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    123,
						Email: "user@example.com",
					}, nil
				},
				ActiveOfferByPublicIDFunc: func(ctx context.Context, id string) (db.Offer, error) {
					return db.Offer{
						ID:       1,
						PriceUsd: sql.NullFloat64{Float64: 9.99, Valid: true},
					}, nil
				},
				InsertPurchaseFunc: func(ctx context.Context, arg db.InsertPurchaseParams) error {
					return nil
				},
				GeneratePurchaseIDFunc: func() string {
					return "purchase-123"
				},
				CreateNowpaymentsInvoiceFunc: func(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
					return &nowpayments.CreateInvoiceResponse{
						InvoiceURL: "https://payments.example.com/invoice/123",
					}, nil
				},
				PublicURLFunc: func() string {
					return "https://example.com"
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-123",
					ReturnPath: "/user/space",
				},
			},
			want: &model.CreatePaymentLinkPayload{
				RedirectURL: "https://payments.example.com/invoice/123",
				Token:       nil, // authenticated user doesn't get token
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UserByIDCalls()))
				require.Equal(t, 1, len(mockEnv.ActiveOfferByPublicIDCalls()))
				require.Equal(t, 1, len(mockEnv.InsertPurchaseCalls()))
				require.Equal(t, 1, len(mockEnv.CreateNowpaymentsInvoiceCalls()))

				// Verify offer lookup
				require.Equal(t, "offer-123", mockEnv.ActiveOfferByPublicIDCalls()[0].ID)

				// Verify purchase insertion
				purchase := mockEnv.InsertPurchaseCalls()[0].Arg
				require.Equal(t, "purchase-123", purchase.ID)
				require.Equal(t, "user@example.com", purchase.Email)
				require.Equal(t, int64(1), purchase.OfferID)
				require.Equal(t, 9.99, purchase.PriceUsd)
				require.Equal(t, "nowpayments", purchase.PaymentProvider)

				// Verify invoice creation
				invoice := mockEnv.CreateNowpaymentsInvoiceCalls()[0].Params
				require.Equal(t, 9.99, invoice.PriceAmount)
				require.Equal(t, "usd", invoice.PriceCurrency)
				require.Equal(t, "purchase-123", invoice.OrderID)
				require.Contains(t, invoice.SuccessURL, "payment_result=success")
				require.Contains(t, invoice.CancelURL, "payment_result=cancel")
			},
		},
		{
			name: "successful payment link creation - unauthenticated user with email",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, nil // unauthenticated
				},
				UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
					return db.User{}, sql.ErrNoRows // user doesn't exist, allow creation
				},
				ActiveOfferByPublicIDFunc: func(ctx context.Context, id string) (db.Offer, error) {
					return db.Offer{
						ID:       1,
						PriceUsd: sql.NullFloat64{Float64: 19.99, Valid: true},
					}, nil
				},
				InsertPurchaseFunc: func(ctx context.Context, arg db.InsertPurchaseParams) error {
					return nil
				},
				GeneratePurchaseIDFunc: func() string {
					return "purchase-456"
				},
				GenerateHotAuthTokenFunc: func(ctx context.Context, data appmodel.HotAuthToken) (string, error) {
					return "hot-auth-token-123", nil
				},
				StorePurchaseTokenFunc: func(ctx context.Context, data appmodel.PurchaseToken) (string, error) {
					return "purchase-token-456", nil
				},
				CreateNowpaymentsInvoiceFunc: func(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
					return &nowpayments.CreateInvoiceResponse{
						InvoiceURL: "https://payments.example.com/invoice/456",
					}, nil
				},
				PublicURLFunc: func() string {
					return "https://example.com/"
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-456",
					ReturnPath: "/user/space",
					Email:      stringPtr("newuser@example.com"),
				},
			},
			want: &model.CreatePaymentLinkPayload{
				RedirectURL: "https://payments.example.com/invoice/456",
				Token:       stringPtr("purchase-token-456"),
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 0, len(mockEnv.CurrentUserTokenCalls())) // not called for unauthenticated with email
				require.Equal(t, 1, len(mockEnv.UserByEmailCalls()))
				require.Equal(t, 1, len(mockEnv.GenerateHotAuthTokenCalls()))
				require.Equal(t, 1, len(mockEnv.StorePurchaseTokenCalls()))

				// Verify hot auth token generation
				hotAuthData := mockEnv.GenerateHotAuthTokenCalls()[0].Data
				require.Equal(t, "newuser@example.com", hotAuthData.Email)

				// Verify purchase token storage
				purchaseTokenData := mockEnv.StorePurchaseTokenCalls()[0].Data
				require.Equal(t, "purchase-456", purchaseTokenData.PurchaseID)

				// Verify success URL contains hot auth token
				invoice := mockEnv.CreateNowpaymentsInvoiceCalls()[0].Params
				require.Contains(t, invoice.SuccessURL, "hat=hot-auth-token-123")
			},
		},
		{
			name: "error - unauthenticated without email",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, nil // unauthenticated
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-123",
					ReturnPath: "/user/space",
					Email:      nil, // no email provided
				},
			},
			want:    &model.ErrorPayload{Message: "unauthenticated"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.ActiveOfferByPublicIDCalls()))
			},
		},
		{
			name: "error - existing user with email requires sign in",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, nil // unauthenticated
				},
				UserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
					return db.User{
						ID:    999,
						Email: "existing@example.com",
					}, nil // user exists
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-123",
					ReturnPath: "/user/space",
					Email:      stringPtr("existing@example.com"),
				},
			},
			want:    &model.ErrorPayload{Message: "sign_in_required"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.UserByEmailCalls()))
				require.Equal(t, 0, len(mockEnv.ActiveOfferByPublicIDCalls()))
			},
		},
		{
			name: "error - offer not found",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    123,
						Email: "user@example.com",
					}, nil
				},
				ActiveOfferByPublicIDFunc: func(ctx context.Context, id string) (db.Offer, error) {
					return db.Offer{}, sql.ErrNoRows
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "nonexistent-offer",
					ReturnPath: "/user/space",
				},
			},
			want:    &model.ErrorPayload{Message: "offer_not_found"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.ActiveOfferByPublicIDCalls()))
				require.Equal(t, 0, len(mockEnv.InsertPurchaseCalls()))
			},
		},
		{
			name: "error - offer has no price",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    123,
						Email: "user@example.com",
					}, nil
				},
				ActiveOfferByPublicIDFunc: func(ctx context.Context, id string) (db.Offer, error) {
					return db.Offer{
						ID:       1,
						PriceUsd: sql.NullFloat64{Valid: false}, // no price
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "free-offer",
					ReturnPath: "/user/space",
				},
			},
			want:    &model.ErrorPayload{Message: "offer_not_found"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.ActiveOfferByPublicIDCalls()))
				require.Equal(t, 0, len(mockEnv.InsertPurchaseCalls()))
			},
		},
		{
			name: "error - current user token error",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("token service error")
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-123",
					ReturnPath: "/user/space",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentUserTokenCalls()))
			},
		},
		{
			name: "error - user lookup error",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-123",
					ReturnPath: "/user/space",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.UserByIDCalls()))
			},
		},
		{
			name: "error - nowpayments invoice creation failed",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{
						ID:    123,
						Email: "user@example.com",
					}, nil
				},
				ActiveOfferByPublicIDFunc: func(ctx context.Context, id string) (db.Offer, error) {
					return db.Offer{
						ID:       1,
						PriceUsd: sql.NullFloat64{Float64: 9.99, Valid: true},
					}, nil
				},
				InsertPurchaseFunc: func(ctx context.Context, arg db.InsertPurchaseParams) error {
					return nil
				},
				GeneratePurchaseIDFunc: func() string {
					return "purchase-123"
				},
				CreateNowpaymentsInvoiceFunc: func(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
					return nil, errors.New("payment service unavailable")
				},
				PublicURLFunc: func() string {
					return "https://example.com"
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-123",
					ReturnPath: "/user/space",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CreateNowpaymentsInvoiceCalls()))
			},
		},
		{
			name: "purchase status is set to pending",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{ID: 1, Email: "test@example.com"}, nil
				},
				ActiveOfferByPublicIDFunc: func(ctx context.Context, id string) (db.Offer, error) {
					return db.Offer{
						ID:       1,
						PriceUsd: sql.NullFloat64{Valid: true, Float64: 10.0},
					}, nil
				},
				InsertPurchaseFunc: func(ctx context.Context, arg db.InsertPurchaseParams) error {
					require.Equal(t, "pending", arg.Status, "Status should be set to 'pending'")
					require.Equal(t, "test@example.com", arg.Email)
					require.Equal(t, int64(1), arg.OfferID)
					require.Equal(t, "nowpayments", arg.PaymentProvider)
					require.Equal(t, "[]", arg.PaymentData)
					require.Equal(t, 10.0, arg.PriceUsd)
					return nil
				},
				GeneratePurchaseIDFunc: func() string {
					return "purchase-123"
				},
				CreateNowpaymentsInvoiceFunc: func(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
					return &nowpayments.CreateInvoiceResponse{
						InvoiceURL: "https://payments.example.com/invoice/123",
					}, nil
				},
				PublicURLFunc: func() string {
					return "https://example.com"
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.CreatePaymentLinkInput{
					OfferID:    "offer-123",
					ReturnPath: "/user/space",
				},
			},
			want: &model.CreatePaymentLinkPayload{
				RedirectURL: "https://payments.example.com/invoice/123",
				Token:       nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createpaymentlink.Resolve(tt.args.ctx, tt.env, tt.args.req)
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

func stringPtr(s string) *string {
	return &s
}