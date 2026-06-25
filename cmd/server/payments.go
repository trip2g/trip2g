package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"trip2g/internal/appreq"
	"trip2g/internal/boosty"
	"trip2g/internal/boostyjobs"
	"trip2g/internal/case/signinbypurchasetoken"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/nowpayments"
	"trip2g/internal/patreon"
	"trip2g/internal/patreonjobs"

	"github.com/valyala/fasthttp"
)

// constructPatreon builds the Patreon client manager (no DB). The credential
// scan, webhook registration, and refresh background jobs (all of which write)
// live in startPatreonWriters (Block B).
func (a *app) constructPatreon() {
	a.patreonClientManager = patreon.NewClientManager(a)
}

func (a *app) startPatreonWriters(ctx context.Context) {
	var err error

	a.PatreonJobs, err = patreonjobs.New(ctx, a, a.config.PatreonJobsConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create Patreon IO: %w", err))
	}
}

// constructBoosty builds the Boosty client manager (no DB). The credential
// scan and refresh background jobs (which write) live in startBoostyWriters
// (Block B).
func (a *app) constructBoosty() {
	a.boostyClientManager = boosty.NewClientManager(a)
}

func (a *app) startBoostyWriters(ctx context.Context) {
	var err error

	a.BoostyJobs, err = boostyjobs.New(ctx, a, a.config.BoostyJobsConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create Boosty IO: %w", err))
	}
}

func (a *app) PatreonClientByID(ctx context.Context, credentialsID int64) (patreon.Client, error) {
	env, err := getEnvOrDefault[patreon.ClientManagerEnv](ctx, a)
	if err != nil {
		return nil, fmt.Errorf("failed to get Patreon client manager environment: %w", err)
	}

	client, err := a.patreonClientManager.Get(ctx, env, credentialsID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Patreon client: %w", err)
	}

	return client, nil
}

func (a *app) UpdateBoostyCredentials(ctx context.Context, args db.UpdateBoostyCredentialsParams) (db.BoostyCredential, error) {
	a.boostyClientManager.Reset(ctx, args.ID)

	return a.WriteQueries.UpdateBoostyCredentials(ctx, args)
}

func (a *app) BoostyClientByCredentialsID(ctx context.Context, credentialID int64) (boosty.Client, error) {
	env, err := getEnvOrDefault[boosty.ClientManagerEnv](ctx, a)
	if err != nil {
		return nil, err
	}

	return a.boostyClientManager.Get(ctx, env, credentialID)
}

func (a *app) CreateNowpaymentsInvoice(params nowpayments.CreateInvoiceParams) (*nowpayments.CreateInvoiceResponse, error) {
	return a.nowpaymentsClient.CreateInvoice(params)
}

const purchaseAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (a *app) GeneratePurchaseID() string {
	const length = 8

	result := make([]byte, length)

	for i := range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(purchaseAlphabet))))
		if err != nil {
			panic(err)
		}

		result[i] = purchaseAlphabet[n.Int64()]
	}

	return string(result)
}

func (a *app) NowpaymentsIPNSecret() string {
	return a.config.NowpaymentsIPNKey
}

func (a *app) StorePurchaseToken(ctx context.Context, data model.PurchaseToken) (string, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return "", err
	}

	return a.purchaseTokenManager.Store(req.Req, data)
}

func (a *app) ExtractPurchaseTokenIDs(ctx context.Context) ([]string, error) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, err
	}

	tokens, err := a.purchaseTokenManager.Extract(req.Req)
	if err != nil {
		return nil, fmt.Errorf("failed to extract purchase tokens: %w", err)
	}

	ids := make([]string, len(tokens))

	for i, token := range tokens {
		ids[i] = token.PurchaseID
	}

	return ids, nil
}

func (a *app) NotifyPuchaseUpdated(email string) {
}

func (a *app) handlePurchaseTokens(ctx *fasthttp.RequestCtx) bool {
	purchaseTokens, _ := a.purchaseTokenManager.Extract(ctx)
	if len(purchaseTokens) > 0 {
		processed, err := signinbypurchasetoken.Resolve(ctx, a, purchaseTokens)
		if err != nil {
			a.log.Warn("failed to resolve purchase token", "error", err)
		} else if processed {
			err = a.purchaseTokenManager.Delete(ctx)
			if err != nil {
				a.log.Warn("failed to delete purchase token", "error", err)
			}
		}
	}

	return false
}
