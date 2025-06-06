package processnowpaymentsipn

import (
	"context"
	"database/sql"
	json "encoding/json"
	"fmt"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/nowpayments"
)

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	NowpaymentsIPNSecret() string
	Now() time.Time
	Logger() logger.Logger
	PurchaseByID(ctx context.Context, id string) (db.Purchase, error)
	OfferByID(ctx context.Context, id int64) (db.Offer, error)
	UpdatePurchaseStatus(ctx context.Context, params db.UpdatePurchaseStatusParams) error
	ListSubgraphsByOfferID(ctx context.Context, offerID int64) ([]db.Subgraph, error)
	UserByEmail(ctx context.Context, email string) (db.User, error)
	InsertUser(ctx context.Context, email string) (db.User, error)
	CountUserSubgraphAccessByPurchaseID(ctx context.Context, purchaseID string) (int64, error)
	CreateUserSubgraphAccess(ctx context.Context, params db.CreateUserSubgraphAccessParams) (db.UserSubgraphAccess, error)
	NotifyPuchaseUpdated(email string)
}

type Response struct {
	Success bool
}

func Resolve(ctx context.Context, env Env, req nowpayments.IPNRequest) (*Response, error) {
	purchase, err := env.PurchaseByID(ctx, req.OrderID)
	if err != nil {
		if db.IsNoFound(err) {
			env.Logger().Info("purchase not found", "order_id", req.OrderID)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get purchase by ID: %w", err)
	}

	err = appendState(ctx, env, req, &purchase)
	if err != nil {
		return nil, fmt.Errorf("failed to append state: %w", err)
	}

	env.Logger().Info("purchase updated", "order_id", req.OrderID, "status", req.PaymentStatus)

	if req.PaymentStatus == nowpayments.PaymentStatusConfirmed { //nolint:nestif // I don't know how to avoid this nesting
		accessCount, countErr := env.CountUserSubgraphAccessByPurchaseID(ctx, purchase.ID)
		if countErr != nil {
			return nil, fmt.Errorf("failed to count user subgraph access by purchase ID: %w", countErr)
		}

		// not granted yet
		if accessCount == 0 {
			grantErr := grantAccesses(ctx, env, &purchase)
			if grantErr != nil {
				return nil, fmt.Errorf("failed to grant accesses: %w", grantErr)
			}
		} else {
			env.Logger().Warn("access already granted", "purchase_id", purchase.ID)
		}
	}

	env.NotifyPuchaseUpdated(purchase.Email)

	response := Response{
		Success: true,
	}

	return &response, nil
}

func grantAccesses(ctx context.Context, env Env, purchase *db.Purchase) error {
	user, err := env.UserByEmail(ctx, purchase.Email)
	if err != nil {
		if db.IsNoFound(err) {
			env.Logger().Info("user not found, creating new user", "email", purchase.Email)

			user, err = env.InsertUser(ctx, purchase.Email)
			if err != nil {
				return fmt.Errorf("failed to insert user: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get user by email: %w", err)
		}
	}

	subgraphs, err := env.ListSubgraphsByOfferID(ctx, purchase.OfferID)
	if err != nil {
		return fmt.Errorf("failed to list subgraphs by offer ID: %w", err)
	}

	offer, err := env.OfferByID(ctx, purchase.OfferID)
	if err != nil {
		return fmt.Errorf("failed to get offer by ID: %w", err)
	}

	expiresAt := sql.NullTime{}
	if offer.Lifetime != nil {
		lifetime, lifetimeErr := offer.Lifetime.Duration()
		if lifetimeErr != nil {
			return fmt.Errorf("failed to get lifetime duration: %w", lifetimeErr)
		}

		expiresAt.Time = env.Now().Add(lifetime)
	}

	for _, subgraph := range subgraphs {
		accessParams := db.CreateUserSubgraphAccessParams{
			UserID:     user.ID,
			SubgraphID: subgraph.ID,
			PurchaseID: purchase.ID,
			ExpiresAt:  expiresAt,
		}

		_, err = env.CreateUserSubgraphAccess(ctx, accessParams)
		if err != nil {
			return fmt.Errorf("failed to create user subgraph access: %w", err)
		}
	}

	return nil
}

func appendState(ctx context.Context, env Env, req nowpayments.IPNRequest, purchase *db.Purchase) error {
	var ipns []json.RawMessage

	err := json.Unmarshal([]byte(purchase.PaymentData), &ipns)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payment data: %w", err)
	}

	rawReq, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	ipns = append(ipns, rawReq)

	newIPNS, err := json.Marshal(ipns)
	if err != nil {
		return fmt.Errorf("failed to marshal new IPN: %w", err)
	}

	purchaseParams := db.UpdatePurchaseStatusParams{
		ID:          purchase.ID,
		Status:      string(req.PaymentStatus),
		PaymentData: string(newIPNS),
	}

	err = env.UpdatePurchaseStatus(ctx, purchaseParams)
	if err != nil {
		return fmt.Errorf("failed to update purchase status: %w", err)
	}

	return nil
}
