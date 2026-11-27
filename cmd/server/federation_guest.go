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
	"trip2g/internal/case/federationleave"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

func (a *app) FederationGuestPeer(_ context.Context, kb *model.MCPFederationNote) (model.FederationPeer, error) {
	return a.outboundPeer(kb, 0)
}

func (a *app) VerifyGuestHandoff(ctx context.Context, token string) (mcp.GuestHandoffClaims, error) {
	return mcp.VerifyGuestHandoff(ctx, a, token)
}

func (a *app) SpendFederationGuestID(id string, expiresAt time.Time) bool {
	return a.guestSpent.SpendFederatedAssetLink(id, expiresAt)
}

func (a *app) GuestTickets() *federationguest.Tickets {
	return &a.guestTickets
}

// FederationGuestUser finds or creates the synthetic user of the pair (edge,
// reader). The user has no email and no Telegram id, so no sign-in path can
// reach it. Two first handoffs at once insert two users; the table's unique key
// keeps one row, and the loser deletes the user it created.
func (a *app) FederationGuestUser(ctx context.Context, kid, reader string) (int64, error) {
	find := db.FederationGuestUserParams{Kid: kid, ReaderHash: reader}

	user, err := a.Queries.FederationGuestUser(ctx, find)
	if err == nil {
		return user.ID, nil
	}
	if !db.IsNoFound(err) {
		return 0, fmt.Errorf("find federation guest: %w", err)
	}

	created, err := a.WriteQueries.InsertFederationGuestUser(ctx)
	if err != nil {
		return 0, fmt.Errorf("insert federation guest user: %w", err)
	}

	insert := db.InsertFederationGuestParams{Kid: kid, ReaderHash: reader, UserID: created.ID}
	inserted, err := a.WriteQueries.InsertFederationGuest(ctx, insert)
	if err != nil {
		return 0, fmt.Errorf("insert federation guest: %w", err)
	}
	if inserted == 1 {
		return created.ID, nil
	}

	err = a.WriteQueries.DeleteUnclaimedFederationGuestUser(ctx, created.ID)
	if err != nil {
		return 0, fmt.Errorf("delete the user of a lost race: %w", err)
	}

	user, err = a.Queries.FederationGuestUser(ctx, find)
	if err != nil {
		return 0, fmt.Errorf("find federation guest after a race: %w", err)
	}
	return user.ID, nil
}

// validateFederationGuest ends a guest session whose key is gone, was replaced
// or no longer accepts people. It runs on every request, so revoking the key or
// switching the setting off takes effect on the guest's next click.
func (a *app) validateFederationGuest(ctx context.Context, data *usertoken.Data) error {
	secret, ok, err := a.FederationSecretByKID(ctx, data.Kid)
	if err != nil {
		return fmt.Errorf("federation secret by kid: %w", err)
	}
	if !ok {
		return fmt.Errorf("edge %q is revoked: %w", data.Kid, usertoken.ErrSessionEnded)
	}
	if secret.ID != data.Key {
		return fmt.Errorf("edge %q is now another key: %w", data.Kid, usertoken.ErrSessionEnded)
	}
	if !secret.AcceptGuests {
		return fmt.Errorf("edge %q does not accept people: %w", data.Kid, usertoken.ErrSessionEnded)
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
	for _, allowed := range []string{"/_system/graphql", "/graphql", federationhandoff.Path, federationguest.Path, federationleave.Path} {
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
