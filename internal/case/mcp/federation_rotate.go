package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

// federationSecretBytes is what a federation key must be. The asking side
// generates it, so the base checks the shape rather than trusting the peer to
// have got it right.
const federationSecretBytes = 32

// rotateSecretToolName mirrors model.RotateSecretTool so the dispatch table and
// the client cannot drift apart.
const rotateSecretToolName = model.RotateSecretTool

// grantedScopeToolName mirrors model.GrantedScopeTool for the same reason.
const grantedScopeToolName = model.GrantedScopeTool

// handleRotateSecret replaces the calling pairing's shared key with the one it
// carries.
//
// The pairing is whichever one signed the request — read from the verified JWT,
// never from an argument — so a peer can only ever rotate its own key. There is
// no way to name another kid and no code path that would honour it.
//
// It is reachable only by a federation-authenticated caller: to anyone else the
// tool does not exist, which is also why it is absent from tools/list. The same
// answer for "not a peer" and "no such method" is deliberate; a caller who is
// not in the federation learns nothing about what this endpoint can do.
func handleRotateSecret(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	auth, ok := federationAuthFromContext(ctx)
	if !ok {
		return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+rotateSecretToolName)
	}

	// The key is in the arguments, and arguments are only as trustworthy as the
	// signature that covers them. A peer that does not bind its body is one
	// whose proposed key anything on the connection could have replaced inside
	// the 30-second window, so this refuses rather than rotating to a value of
	// unknown origin.
	if !auth.BodyBound {
		return errorResponse(id, ErrCodeInvalidRequest, "rotation requires a request whose body is covered by the signature")
	}

	args, errResp := unmarshalArgs[RotateSecretArguments](argsRaw, id, rotateSecretToolName)
	if errResp != nil {
		return *errResp
	}

	secret, err := hex.DecodeString(args.SecretHex)
	if err != nil || len(secret) != federationSecretBytes {
		return errorResponse(id, ErrCodeInvalidParams, "secret_hex must be 64 hex characters")
	}
	if isZeroBytes(secret) {
		return errorResponse(id, ErrCodeInvalidParams, "secret_hex must not be all zeroes")
	}

	row, err := env.FederationSecretByID(ctx, auth.SecretID)
	if err != nil && !db.IsNoFound(err) {
		return errorResponse(id, ErrCodeInternal, "Rotation failed: "+err.Error())
	}
	// Revoked or gone between verification and here: the same answer a stranger
	// gets, because by now this caller is one.
	if db.IsNoFound(err) || row.RevokedAt != nil {
		return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+rotateSecretToolName)
	}

	current, err := env.DecryptData(row.SecretCrypt)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Rotation failed: "+err.Error())
	}

	// Already applied. A peer whose response was lost retries with the same key,
	// and answering success without shifting the previous one again is what
	// makes that retry safe to repeat.
	if subtle.ConstantTimeCompare(current, secret) == 1 {
		return successResponse(id, textToolResult("rotated"))
	}

	encrypted, err := env.EncryptData(secret)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Rotation failed: "+err.Error())
	}

	rotateParams := db.RotateFederationSecretParams{
		NewSecretCrypt:      encrypted,
		ID:                  row.ID,
		ExpectedSecretCrypt: row.SecretCrypt,
	}

	_, err = env.RotateFederationSecret(ctx, rotateParams)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Rotation failed: "+err.Error())
	}

	env.AuditLogger().Info("federation secret rotated", "kid", auth.KID, "secretID", row.ID)

	return successResponse(id, textToolResult("rotated"))
}

func isZeroBytes(value []byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}

// handleGrantedScope answers the calling pairing with the subgraphs it may see.
//
// The answer is already in hand: verifyInbound resolved it to decide what this
// request may read, so this returns that and touches nothing. It is the same
// list the filter uses, which is the point — a peer asking what it is allowed to
// see has to be told what it is actually allowed to see, not a second opinion
// assembled separately.
//
// Empty is a real answer and not an error: a kid with no rows granted is
// anonymous-equivalent, and saying so is what distinguishes "scoped to nothing"
// from "nothing matched your query" — the two failures that look identical from
// the asking side.
func handleGrantedScope(ctx context.Context, _ Env, id any, _ json.RawMessage) Response {
	auth, ok := federationAuthFromContext(ctx)
	if !ok {
		return errorResponse(id, ErrCodeMethodNotFound, "Method not found: "+grantedScopeToolName)
	}

	payload := GrantedScopePayload{Subgraphs: auth.AllowedSubgraphs}
	if payload.Subgraphs == nil {
		payload.Subgraphs = []string{}
	}

	return successResponse(id, structuredToolResult(describeScope(payload.Subgraphs), payload))
}

func describeScope(subgraphs []string) string {
	if len(subgraphs) == 0 {
		return "no subgraphs granted"
	}
	return strings.Join(subgraphs, ", ")
}
