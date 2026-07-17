package graph

import (
	"context"
	"errors"
	"fmt"

	"trip2g/internal/case/checkapikey"
)

var errReaderMovesForbidden = errors.New("readerMoves: admin session or instance API key required")

// readerMovesAuth authorizes a readerMoves subscription. Stricter than
// noteChanges: live reader movement is admin-surface data, so only the admin
// session and instance API key lanes are accepted. Webhook shortapitokens and
// ordinary user sessions are rejected — there is no per-note ACL that could
// scope a movement stream.
func readerMovesAuth(ctx context.Context, env noteChangesAuthEnv) error {
	token, err := env.CurrentUserToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user token: %w", err)
	}
	if token.IsAdmin() {
		return nil
	}

	apiKey, err := checkapikey.Resolve(ctx, env, "reader_moves")
	if err != nil {
		if errors.Is(err, checkapikey.ErrMissingKey) {
			return errReaderMovesForbidden
		}
		return err
	}
	// Virtual keys (ID 0: webhook shortapitoken) are not instance credentials.
	if apiKey.ID == 0 {
		return errReaderMovesForbidden
	}
	return nil
}
