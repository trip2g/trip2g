package createinboundfederationsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createinboundfederationsecret_test . Env

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/db"
	"trip2g/internal/federationkey"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

// mcpPath is where a peer speaks to this instance. Part of the address handed
// over rather than something the other operator appends, because appending it is
// a step that gets forgotten and the failure lands minutes later as a 404.
const mcpPath = "/_system/mcp"

type Env interface {
	InsertFederationSecret(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	EncryptData(plaintext []byte) ([]byte, error)
	PublicURL() string
}

type Input = model.CreateInboundFederationSecretInput
type Payload = model.CreateInboundFederationSecretOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Kid, ozzo.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	secret, err := resolveSecret(input.SecretHex)
	if err != nil {
		return &model.ErrorPayload{Message: err.Error()}, nil //nolint:nilerr // intentional: convert internal error to user-facing payload
	}

	encrypted, err := env.EncryptData(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	params := db.InsertFederationSecretParams{
		Kid:         input.Kid,
		SecretCrypt: encrypted,
		KbUrl:       nil,
		Description: input.Description,
		CreatedBy:   int64(token.ID),
	}

	row, err := env.InsertFederationSecret(ctx, params)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return &model.ErrorPayload{Message: "Federation secret with this kid already exists"}, nil
		}
		return nil, fmt.Errorf("failed to insert federation secret: %w", err)
	}

	// The key is assembled here rather than by whoever reads the payload: this
	// side is the only one that knows its own address, and an operator asked to
	// supply it types the wrong one — a hostname without the /_system/mcp path,
	// or the address they reach the admin on rather than the one a peer can.
	handover := federationkey.Handover{
		KID:       row.Kid,
		KBURL:     strings.TrimRight(env.PublicURL(), "/") + mcpPath,
		SecretHex: hex.EncodeToString(secret),
		KBID:      suggestedKBID(env.PublicURL()),
	}

	key, err := federationkey.Encode(handover)
	if err != nil {
		return nil, err
	}

	payload := model.CreateInboundFederationSecretPayload{
		ID:        row.ID,
		Kid:       row.Kid,
		SecretHex: handover.SecretHex,
		Key:       key,
	}

	return &payload, nil
}

// resolveSecret returns the 32-byte secret to store: caller-supplied hex if
// provided, otherwise crypto/rand. Hex must decode to exactly 32 bytes.
func resolveSecret(provided *string) ([]byte, error) {
	if provided == nil || *provided == "" {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, fmt.Errorf("failed to generate random secret: %w", err)
		}
		return secret, nil
	}

	decoded, err := hex.DecodeString(*provided)
	if err != nil {
		return nil, fmt.Errorf("secretHex is not valid hex: %w", err)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("secretHex must decode to exactly 32 bytes, got %d", len(decoded))
	}
	return decoded, nil
}

// suggestedKBID proposes the slug the other side will address this base by. Its
// hostname, which is also what trip2g falls back to when a KB-note names no
// mcp_federation_kb_id — so the suggestion and the default agree.
func suggestedKBID(publicURL string) string {
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	return parsed.Hostname()
}
