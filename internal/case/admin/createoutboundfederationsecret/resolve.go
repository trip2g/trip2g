package createoutboundfederationsecret

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createoutboundfederationsecret_test . Env

import (
	"context"
	"encoding/hex"
	"fmt"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"trip2g/internal/case/admin/rotatefederationsecret"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	InsertFederationSecret(ctx context.Context, arg db.InsertFederationSecretParams) (db.FederationSecret, error)
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	rotatefederationsecret.Env
}

type Input = model.CreateOutboundFederationSecretInput
type Payload = model.CreateOutboundFederationSecretOrErrorPayload

func validateRequest(r *Input) *model.ErrorPayload {
	return model.NewOzzoError(ozzo.ValidateStruct(r,
		ozzo.Field(&r.Kid, ozzo.Required),
		ozzo.Field(&r.SecretHex, ozzo.Required),
		ozzo.Field(&r.KbURL, ozzo.Required),
	))
}

func Resolve(ctx context.Context, env Env, input Input) (Payload, error) {
	errPayload := validateRequest(&input)
	if errPayload != nil {
		return errPayload, nil
	}

	secret, err := hex.DecodeString(input.SecretHex)
	if err != nil || len(secret) != 32 { //nolint:nilerr // intentional: convert decode error to user-facing payload
		return &model.ErrorPayload{Message: "secretHex must be exactly 32 bytes (64 hex characters)"}, nil
	}

	// Rotation before the row, not after it. The key that arrived out of band is
	// only ever used to ask the peer to stop honouring it, and if the peer will
	// not — unreachable, or too old to know the call — nothing is stored at all.
	// A row left holding the handed-over key would be a link that works while
	// quietly resting on the credential this exists to retire.
	peerSecret := secret
	rotated := false
	if input.Rotate == nil || *input.Rotate {
		fresh, mintErr := rotatefederationsecret.NewSecret()
		if mintErr != nil {
			return nil, mintErr
		}

		peer := appmodel.FederationPeer{KBURL: input.KbURL, KID: input.Kid, Secret: secret}

		// Nothing is stored unless the peer confirmed. At install there is no
		// link to protect yet, so silence and refusal lead to the same place —
		// ask the other operator for a fresh handover and try again, which costs
		// them one mutation. That is cheaper than a row resting on the key that
		// travelled through a chat.
		proposeErr := rotatefederationsecret.Propose(ctx, env, peer, fresh)
		if proposeErr != nil {
			//nolint:nilerr // intentional: a peer that will not confirm is an answer an operator acts on, not a fault
			return &model.ErrorPayload{Message: proposeErr.Error()}, nil
		}
		peerSecret = fresh
		rotated = true
	}

	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current admin user token: %w", err)
	}

	encrypted, err := env.EncryptData(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	kbURL := input.KbURL
	params := db.InsertFederationSecretParams{
		Kid:         input.Kid,
		SecretCrypt: encrypted,
		KbUrl:       &kbURL,
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

	// Two writes rather than an insert that knows about rotation: the row starts
	// holding what the peer used to accept, and the same statement an operator's
	// rotation uses moves it forward. One definition of what a rotated row looks
	// like, reached by both paths.
	if rotated {
		rotatedCrypt, encErr := env.EncryptData(peerSecret)
		if encErr != nil {
			return nil, fmt.Errorf("failed to encrypt rotated secret: %w", encErr)
		}

		rotateParams := db.RotateFederationSecretParams{
			NewSecretCrypt:      rotatedCrypt,
			ID:                  row.ID,
			ExpectedSecretCrypt: row.SecretCrypt,
		}

		_, err = env.RotateFederationSecret(ctx, rotateParams)
		if err != nil {
			return nil, fmt.Errorf("failed to store rotated secret: %w", err)
		}

		confirmPeer := appmodel.FederationPeer{
			KBURL:  input.KbURL,
			KID:    input.Kid,
			Secret: peerSecret,
		}
		rotatefederationsecret.Confirm(ctx, env, row.ID, row.SecretCrypt, confirmPeer)
	}

	payload := model.CreateOutboundFederationSecretPayload{
		ID:  row.ID,
		Kid: row.Kid,
	}

	return &payload, nil
}
