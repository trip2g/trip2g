package processnowpaymentsipn

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/nowpayments"
)

type successfulPaymentEnv struct {
	createdAccesses []db.CreateUserSubgraphAccessParams
	notifications   int
}

func (e *successfulPaymentEnv) NowpaymentsIPNSecret() string {
	return "secret"
}

func (e *successfulPaymentEnv) Now() time.Time {
	return time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
}

func (e *successfulPaymentEnv) Logger() logger.Logger {
	return &logger.DummyLogger{}
}

func (e *successfulPaymentEnv) PurchaseByID(_ context.Context, id string) (db.Purchase, error) {
	return db.Purchase{
		ID:          id,
		PaymentData: `[]`,
		OfferID:     42,
		Email:       "buyer@example.com",
	}, nil
}

func (e *successfulPaymentEnv) OfferByID(_ context.Context, id int64) (db.Offer, error) {
	return db.Offer{ID: id}, nil
}

func (e *successfulPaymentEnv) UpdatePurchaseStatus(_ context.Context, _ db.UpdatePurchaseStatusParams) error {
	return nil
}

func (e *successfulPaymentEnv) ListSubgraphsByOfferID(_ context.Context, _ int64) ([]db.Subgraph, error) {
	return []db.Subgraph{{ID: 7, Name: "paid"}}, nil
}

func (e *successfulPaymentEnv) UserByEmail(_ context.Context, email string) (db.User, error) {
	return db.User{ID: 9, Email: &email}, nil
}

func (e *successfulPaymentEnv) InsertUserWithEmail(_ context.Context, params db.InsertUserWithEmailParams) (db.User, error) {
	return db.User{ID: 9, Email: &params.Email}, nil
}

func (e *successfulPaymentEnv) CountUserSubgraphAccessByPurchaseID(_ context.Context, _ *string) (int64, error) {
	return 0, nil
}

func (e *successfulPaymentEnv) CreateUserSubgraphAccess(
	_ context.Context,
	params db.CreateUserSubgraphAccessParams,
) (db.UserSubgraphAccess, error) {
	e.createdAccesses = append(e.createdAccesses, params)
	return db.UserSubgraphAccess{
		UserID:     params.UserID,
		SubgraphID: params.SubgraphID,
		PurchaseID: params.PurchaseID,
	}, nil
}

func (e *successfulPaymentEnv) NotifyPuchaseUpdated(_ string) {
	e.notifications++
}

func TestResolveGrantsAccessForEverySuccessfulPaymentStatus(t *testing.T) {
	for _, status := range []nowpayments.PaymentStatus{
		nowpayments.PaymentStatusConfirmed,
		nowpayments.PaymentStatusSending,
		nowpayments.PaymentStatusFinished,
	} {
		t.Run(string(status), func(t *testing.T) {
			require.True(t, status.IsSuccessful(), "the status contract classifies this payment as successful")
			env := &successfulPaymentEnv{}

			response, err := Resolve(context.Background(), env, nowpayments.IPNRequest{
				OrderID:       "purchase-1",
				PaymentStatus: status,
			})

			require.NoError(t, err)
			require.NotNil(t, response)
			require.True(t, response.Success)
			require.Len(t, env.createdAccesses, 1,
				"every successful payment status must grant the purchased subgraph access exactly once")
			require.Equal(t, int64(7), env.createdAccesses[0].SubgraphID)
			require.Equal(t, "purchase-1", *env.createdAccesses[0].PurchaseID)
		})
	}
}
