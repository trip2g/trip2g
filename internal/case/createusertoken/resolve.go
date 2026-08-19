package createusertoken

import (
	"context"
	"fmt"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/personaltoken"
)

const maxActiveTokens = 10

// Instructions is handed to whoever receives the token, token included, so the
// whole text is one copy away from working. That makes it a credential: it is
// returned once, never stored, and never logged. Static on purpose — an
// editable prompt is a separate feature.
const Instructions = `You have been given read access to a trip2g knowledge base.

Connect it as an MCP server over HTTP:

  URL:    %s/_system/mcp
  Header: Authorization: Bearer %s

The token identifies you and carries exactly the subgraphs you were granted —
nothing else is reachable with it. Answer from what you retrieve, and say so
plainly when the base does not cover a question.`

type Env interface {
	CountActiveUserTokensByUserID(ctx context.Context, userID int64) (int64, error)
	GenerateUniqID() string
	InsertUserToken(ctx context.Context, arg db.InsertUserTokenParams) (db.UserToken, error)
	PublicURL() string
}

type Input = model.CreateUserTokenInput
type Payload = model.CreateUserTokenOrErrorPayload

// Resolve mints a personal token for userID. Who that user is — the caller
// themselves or someone an admin picked — is decided by the resolver, so this
// case carries no branch and no notion of the current session.
func Resolve(ctx context.Context, env Env, userID int64, input Input) (Payload, error) {
	count, err := env.CountActiveUserTokensByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active user tokens: %w", err)
	}
	if count >= maxActiveTokens {
		return &model.ErrorPayload{Message: "token_limit_exceeded"}, nil
	}

	plaintext := personaltoken.Generate()
	hash := personaltoken.Hash(plaintext)
	prefix := personaltoken.DisplayPrefix(plaintext)

	var expiresAt *time.Time
	if input.ExpiresInDays != nil {
		t := time.Now().Add(time.Duration(*input.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}

	token, err := env.InsertUserToken(ctx, db.InsertUserTokenParams{
		ID:          env.GenerateUniqID(),
		UserID:      userID,
		Name:        input.Name,
		TokenHash:   hash,
		TokenPrefix: prefix,
		Scope:       "all",
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert user token: %w", err)
	}

	payload := model.CreateUserTokenPayload{
		PlaintextToken: plaintext,
		Token:          &token,
		Instructions:   fmt.Sprintf(Instructions, env.PublicURL(), plaintext),
	}

	return &payload, nil
}
