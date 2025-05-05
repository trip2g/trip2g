package createpaymentlink

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"trip2g/internal/case/processnowpaymentsipn"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/nowpayments"
)

type Env interface {
	PublicURL() string
	CreateNowpaymentsInvoice(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error)
	ActiveOfferByPublicID(ctx context.Context, id string) (db.Offer, error)
	InsertPurchase(ctx context.Context, arg db.InsertPurchaseParams) error
	GeneratePurchaseID() string
	GenerateHotAuthToken(ctx context.Context, data appmodel.HotAuthToken) (string, error)
}

func Resolve(ctx context.Context, env Env, req model.CreatePaymentLinkInput) (model.CreatePaymentLinkOrErrorPayload, error) {
	offer, err := env.ActiveOfferByPublicID(ctx, req.OfferID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "offer_not_found"}, nil
		}

		return nil, fmt.Errorf("failed to get offer: %w", err)
	}

	if !offer.PriceUsd.Valid {
		return &model.ErrorPayload{Message: "offer_not_found"}, nil
	}

	purchaseParams := db.InsertPurchaseParams{
		OfferID: offer.ID,
		Email:   req.Email,

		PaymentProvider: "nowpayments",
		PaymentData:     "[]", // empty array
	}

	err = insertPurchase(ctx, env, &purchaseParams)
	if err != nil {
		return nil, err
	}

	publicURL := strings.TrimRight(env.PublicURL(), "/")
	rawReturnURL := fmt.Sprintf("%s/%s", publicURL, strings.Trim(req.ReturnPath, "/"))

	hotAuthToken, err := env.GenerateHotAuthToken(ctx, appmodel.HotAuthToken{Email: req.Email})
	if err != nil {
		return nil, fmt.Errorf("failed to generate hot auth token: %w", err)
	}

	returnURL, err := url.Parse(rawReturnURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse return URL: %w", err)
	}

	q := returnURL.Query()
	q.Set("hat", hotAuthToken)
	returnURL.RawQuery = q.Encode()
	returnURL.Fragment = "status=success"
	successURL := returnURL.String()

	returnURL.Fragment = "status=cancel"
	cancelURL := returnURL.String()

	invoiceParams := nowpayments.CreateInvoiceParams{
		PriceAmount:      offer.PriceUsd.Float64,
		PriceCurrency:    "usd",
		OrderID:          purchaseParams.ID,
		OrderDescription: "Second brain course",
		IPNCallbackURL:   publicURL + (&processnowpaymentsipn.Endpoint{}).Path(),
		SuccessURL:       successURL,
		CancelURL:        cancelURL,
	}

	invoice, err := env.CreateNowpaymentsInvoice(invoiceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	response := model.CreatePaymentLinkPayload{
		RedirectURL: invoice.InvoiceURL,
	}

	return &response, nil
}

var ErrDuplicatePurchaseID = fmt.Errorf("duplicate purchase ID")

func insertPurchase(ctx context.Context, env Env, params *db.InsertPurchaseParams) error {
	// 16 tryes to insert purchase with unique ID
	for tryCount := 0; tryCount < 16; tryCount++ {
		params.ID = env.GeneratePurchaseID()

		err := env.InsertPurchase(ctx, *params)
		if err != nil {
			if db.IsUniqueViolation(err) {
				continue
			}

			return fmt.Errorf("failed to insert purchase: %w", err)
		}

		return nil
	}

	return ErrDuplicatePurchaseID
}
