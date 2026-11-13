package federationhandoff_test

import (
	"context"

	"trip2g/internal/db"
	"trip2g/internal/logger"
)

// verifyEnv is the peer's side of the edge the tests sign with.
type verifyEnv struct{}

func (verifyEnv) FederationSecretByKID(_ context.Context, kid string) (db.FederationSecret, bool, error) {
	return db.FederationSecret{Kid: kid, SecretCrypt: edgeSecret}, kid == "edge", nil
}

func (verifyEnv) ListFederationSecretSubgraphsByKID(context.Context, string) ([]string, error) {
	return nil, nil
}

func (verifyEnv) DecryptData(data []byte) ([]byte, error) { return data, nil }

func (verifyEnv) ClearFederationSecretPrev(context.Context, db.ClearFederationSecretPrevParams) error {
	return nil
}

func (verifyEnv) Logger() logger.Logger { return &logger.DummyLogger{} }
