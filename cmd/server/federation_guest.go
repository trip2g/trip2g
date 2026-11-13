package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/case/federationguest"
	"trip2g/internal/case/federationhandoff"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

func (a *app) FederationGuestsEnabled() bool {
	return a.config.FederationAcceptGuests
}

func (a *app) FederationGuestPeer(ctx context.Context, kbID string) (model.FederationPeer, error) {
	return a.federationPeer(ctx, kbID)
}

func (a *app) VerifyGuestHandoff(ctx context.Context, token string) (mcp.GuestHandoffClaims, error) {
	return mcp.VerifyGuestHandoff(ctx, a, token)
}

func (a *app) SpendFederationGuestID(id string, expiresAt time.Time) bool {
	return a.guestSpent.SpendFederatedAssetLink(id, expiresAt)
}

func (a *app) GuestTicketSecret() string {
	return a.config.UserToken.Secret
}

// federationGuestCreatedVia names the edge and the reader on the synthetic
// user, which is what view analytics group by. The row has no email and no
// Telegram id, so no sign-in path can reach it.
func federationGuestCreatedVia(kid, reader string) string {
	return "federation:" + kid + ":" + reader
}

func (a *app) FederationGuestUser(ctx context.Context, kid, reader string) (int64, error) {
	createdVia := federationGuestCreatedVia(kid, reader)

	user, err := a.Queries.FederationGuestUser(ctx, createdVia)
	if err == nil {
		return user.ID, nil
	}
	if !db.IsNoFound(err) {
		return 0, fmt.Errorf("find federation guest: %w", err)
	}

	user, err = a.WriteQueries.InsertFederationGuestUser(ctx, createdVia)
	if err != nil {
		return 0, fmt.Errorf("insert federation guest: %w", err)
	}
	return user.ID, nil
}

// validateFederationGuest ends a guest session whose edge is gone or whose
// instance no longer accepts guests. It runs on every request, so revoking the
// key or turning the setting off takes effect on the guest's next click.
func (a *app) validateFederationGuest(ctx context.Context, data *usertoken.Data) error {
	if !a.FederationGuestsEnabled() {
		return fmt.Errorf("guests are not accepted: %w", usertoken.ErrSessionEnded)
	}

	_, ok, err := a.FederationSecretByKID(ctx, data.Kid)
	if err != nil {
		return fmt.Errorf("federation secret by kid: %w", err)
	}
	if !ok {
		return fmt.Errorf("edge %q is revoked: %w", data.Kid, usertoken.ErrSessionEnded)
	}
	return nil
}

// guestScope is what a guest may read: the subgraphs granted to its edge now,
// not when it arrived. A failed lookup grants nothing beyond free notes.
func (a *app) guestScope(ctx context.Context, token *usertoken.Data) []string {
	subgraphs, err := a.ListFederationSecretSubgraphsByKID(ctx, token.Kid)
	if err != nil {
		a.log.Error("federation guest: failed to list edge subgraphs", "kid", token.Kid, "error", err)
		return []string{}
	}
	return subgraphs
}

// guestMayPost names the non-GET paths a guest's request still reaches:
// GraphQL refuses its mutations itself, and the two handoff legs are how a
// guest moves on.
func guestMayPost(path string) bool {
	for _, allowed := range []string{"/_system/graphql", "/graphql", federationhandoff.Path, federationguest.Path} {
		if path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
	}
	return false
}

// denyGuestWrites answers every request a guest makes with a method that can
// change something, outside guestMayPost, with 403.
func (a *app) denyGuestWrites(req *appreq.Request) bool {
	switch string(req.Req.Method()) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	if guestMayPost(req.Path) {
		return false
	}

	if err := appreq.DenyGuest(req.Req); err != nil {
		req.Req.SetStatusCode(http.StatusForbidden)
		req.Req.SetBodyString("403 " + err.Error())
		return true
	}
	return false
}
