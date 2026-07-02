package graph

import (
	"context"
	"errors"
	"fmt"

	"trip2g/internal/case/checkapikey"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/notebus"
)

//go:generate go tool github.com/matryer/moq -out note_changes_acl_mocks_test.go -pkg graph . noteChangesACLEnv noteChangesAuthEnv

// noteChangesACLEnv is the minimal env for per-subscriber ACL filtering of
// note change events.
type noteChangesACLEnv interface {
	LatestNoteViews() *appmodel.NoteViews
	CanReadNote(ctx context.Context, note *appmodel.NoteView) (bool, error)
	Logger() logger.Logger
}

// noteChangesAuthEnv is the env for authorizing a noteChanges subscription.
type noteChangesAuthEnv interface {
	checkapikey.Env
}

// noteChangesAuth authorizes a noteChanges subscription and reports whether
// the subscriber bypasses per-note ACL filtering (the returned bool).
//
// Credential kinds:
//   - instance X-API-Key (real DB row)  -> bypassACL=true (full firehose, admin-created instance credential)
//   - admin session token               -> bypassACL=false (CanReadNote returns true for admin anyway)
//   - webhook Bearer shortapitoken      -> bypassACL=false (CanReadNote enforces read_patterns via appreq.Scoped)
//   - personal user session (non-admin) -> bypassACL=false (CanReadNote scopes by subgraph access)
//   - anonymous                         -> rejected with checkapikey.ErrMissingKey
func noteChangesAuth(ctx context.Context, env noteChangesAuthEnv) (bool, error) {
	apiKey, err := checkapikey.Resolve(ctx, env, "note_changes")
	if err == nil {
		// Only real (DB-backed) instance API keys keep the historical
		// full-firehose contract. Admin bypass and webhook shortapitoken
		// come back as virtual keys with ID 0 and go through per-note ACL.
		return apiKey.ID != 0, nil
	}
	if !errors.Is(err, checkapikey.ErrMissingKey) {
		return false, err
	}

	// No API key: accept a signed-in user session (personal token), scoped by
	// per-note ACL on every emitted change.
	token, tokenErr := env.CurrentUserToken(ctx)
	if tokenErr != nil {
		return false, fmt.Errorf("failed to get current user token: %w", tokenErr)
	}
	if token == nil || token.ID == 0 {
		return false, err // checkapikey.ErrMissingKey
	}
	return false, nil
}

// filterChangesByACL returns only the changes whose note the current
// subscriber is allowed to read (per env.CanReadNote — the same primitive as
// the MCP read path). Fail-closed: a change whose NoteView cannot be resolved
// (e.g. an already-removed note no longer cached) or whose ACL check errors is
// dropped. Always returns a fresh slice — the input batch is shared between
// subscribers and must not be mutated.
func filterChangesByACL(ctx context.Context, env noteChangesACLEnv, changes []notebus.Change) []notebus.Change {
	nvs := env.LatestNoteViews()
	out := make([]notebus.Change, 0, len(changes))
	if nvs == nil {
		return out
	}
	for _, change := range changes {
		nv := nvs.GetByPathID(change.PathID)
		if nv == nil && change.Path != "" {
			nv = nvs.PathMap[change.Path]
		}
		if nv == nil {
			continue
		}
		ok, err := env.CanReadNote(ctx, nv)
		if err != nil {
			env.Logger().Warn("noteChanges: ACL check failed, dropping change", "path", change.Path, "error", err)
			continue
		}
		if ok {
			out = append(out, change)
		}
	}
	return out
}
