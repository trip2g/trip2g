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
	"trip2g/internal/usertoken"
)

type Env interface {
	PublicURL() string
	CreateNowpaymentsInvoice(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error)
	ActiveOfferByPublicID(ctx context.Context, id string) (db.Offer, error)
	InsertPurchase(ctx context.Context, arg db.InsertPurchaseParams) error
	GeneratePurchaseID() string
	GenerateHotAuthToken(ctx context.Context, data appmodel.HotAuthToken) (string, error)
	CurrentUserToken(ctx context.Context) (*usertoken.Data, error)
	UserByID(ctx context.Context, id int64) (db.User, error)
}

func Resolve(ctx context.Context, env Env, req model.CreatePaymentLinkInput) (model.CreatePaymentLinkOrErrorPayload, error) {
	isAuthenticated := false

	if req.Email == nil {
		token, err := env.CurrentUserToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get user token: %w", err)
		}

		if token == nil {
			return &model.ErrorPayload{Message: "unauthenticated"}, nil
		}

		user, err := env.UserByID(ctx, int64(token.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		req.Email = &user.Email

		isAuthenticated = true
	}

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
		Email:   *req.Email,

		PaymentProvider: "nowpayments",
		PaymentData:     "[]", // empty array
	}

	err = insertPurchase(ctx, env, &purchaseParams)
	if err != nil {
		return nil, err
	}

	callbackURLs, err := prepareCallbackURLs(ctx, env, req, isAuthenticated)
	if err != nil {
		return nil, err
	}

	invoiceParams := nowpayments.CreateInvoiceParams{
		PriceAmount:      offer.PriceUsd.Float64,
		PriceCurrency:    "usd",
		OrderID:          purchaseParams.ID,
		OrderDescription: "Second brain course",
		IPNCallbackURL:   callbackURLs.IPNCallbackURL,
		SuccessURL:       callbackURLs.successURL,
		CancelURL:        callbackURLs.cancelURL,
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

type callbackURLs struct {
	IPNCallbackURL string
	successURL     string
	cancelURL      string
}

func prepareCallbackURLs(ctx context.Context, env Env, req model.CreatePaymentLinkInput, isAuthenticated bool) (*callbackURLs, error) {
	publicURL := strings.TrimRight(env.PublicURL(), "/")
	rawReturnURL := fmt.Sprintf("%s/%s", publicURL, strings.Trim(req.ReturnPath, "/"))

	result := callbackURLs{
		IPNCallbackURL: fmt.Sprintf("%s/%s", publicURL, (&processnowpaymentsipn.Endpoint{}).Path()),
	}

	returnURL, err := url.Parse(rawReturnURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse return URL: %w", err)
	}

	// build cancelURL
	q := returnURL.Query()
	q.Set("payment_result", "cancel")
	returnURL.RawQuery = q.Encode()
	result.cancelURL = returnURL.String()

	// build successURL
	if !isAuthenticated {
		hotAuthToken, err := env.GenerateHotAuthToken(ctx, appmodel.HotAuthToken{Email: *req.Email})
		if err != nil {
			return nil, fmt.Errorf("failed to generate hot auth token: %w", err)
		}

		q.Set("hat", hotAuthToken)
	}

	q.Set("payment_result", "success")
	returnURL.RawQuery = q.Encode()
	result.successURL = returnURL.String()

	return &result, nil
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
