package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"

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
		SecretCrypt: encrypted,
		ID:          row.ID,
	}

	err = env.RotateFederationSecret(ctx, rotateParams)
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
