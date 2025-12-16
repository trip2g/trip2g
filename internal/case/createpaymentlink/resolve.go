package createpaymentlink

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"trip2g/internal/case/processnowpaymentsipn"
	"trip2g/internal/case/requestemailsignin"
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
	UserByEmail(ctx context.Context, email string) (db.User, error)
	StorePurchaseToken(ctx context.Context, data appmodel.PurchaseToken) (string, error)

	requestemailsignin.Env
}

type Input = model.CreatePaymentLinkInput
type Payload = model.CreatePaymentLinkOrErrorPayload

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	isAuthenticated := false

	if input.Email == nil { //nolint:nestif // I don't know how to avoid this nesting
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

		if user.Email != nil {
			input.Email = user.Email
		}

		isAuthenticated = true
	} else {
		_, userErr := env.UserByEmail(ctx, *input.Email)
		if userErr != nil {
			if !db.IsNoFound(userErr) {
				return nil, fmt.Errorf("failed to get user by email: %w", userErr)
			}
		} else {
			return &model.ErrorPayload{Message: "sign_in_required"}, nil
		}
	}

	offer, err := env.ActiveOfferByPublicID(ctx, input.OfferID)
	if err != nil {
		if db.IsNoFound(err) {
			return &model.ErrorPayload{Message: "offer_not_found"}, nil
		}

		return nil, fmt.Errorf("failed to get offer: %w", err)
	}

	if offer.PriceUsd == nil {
		return &model.ErrorPayload{Message: "offer_not_found"}, nil
	}

	purchaseParams := db.InsertPurchaseParams{
		OfferID: offer.ID,
		Email:   *input.Email,

		PriceUsd:        *offer.PriceUsd,
		PaymentProvider: "nowpayments",
		PaymentData:     "[]", // empty array
		Status:          "pending",
	}

	err = insertPurchase(ctx, env, &purchaseParams)
	if err != nil {
		return nil, err
	}

	callbackURLs, err := prepareCallbackURLs(ctx, env, input, isAuthenticated)
	if err != nil {
		return nil, err
	}

	invoiceParams := nowpayments.CreateInvoiceParams{
		PriceAmount:      *offer.PriceUsd,
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

	if !isAuthenticated {
		purchaseToken := appmodel.PurchaseToken{
			PurchaseID: purchaseParams.ID,
		}

		rawToken, storeErr := env.StorePurchaseToken(ctx, purchaseToken)
		if storeErr != nil {
			return nil, fmt.Errorf("failed to store purchase token: %w", storeErr)
		}

		response.Token = &rawToken
	}

	return &response, nil
}

type callbackURLs struct {
	IPNCallbackURL string
	successURL     string
	cancelURL      string
}

func prepareCallbackURLs(
	ctx context.Context,
	env Env,
	input model.CreatePaymentLinkInput,
	isAuthenticated bool,
) (*callbackURLs, error) {
	publicURL := strings.TrimRight(env.PublicURL(), "/")
	rawReturnURL := fmt.Sprintf("%s/%s", publicURL, strings.Trim(input.ReturnPath, "/"))

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
		hotAuthToken, hotAuthErr := env.GenerateHotAuthToken(ctx, appmodel.HotAuthToken{Email: *input.Email})
		if hotAuthErr != nil {
			return nil, fmt.Errorf("failed to generate hot auth token: %w", hotAuthErr)
		}

		q.Set("hat", hotAuthToken)
	}

	q.Set("payment_result", "success")
	returnURL.RawQuery = q.Encode()
	result.successURL = returnURL.String()

	return &result, nil
}

var ErrDuplicatePurchaseID = errors.New("duplicate purchase ID")

func insertPurchase(ctx context.Context, env Env, params *db.InsertPurchaseParams) error {
	// 16 tryes to insert purchase with unique ID
	for range 16 {
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
