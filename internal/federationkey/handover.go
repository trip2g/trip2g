// Package federationkey packs everything one side of a federation pairing has to
// tell the other into a single string.
//
// The three values are a set, not three settings: a kid without its secret is
// useless, a secret against the wrong URL authenticates nothing, and an operator
// copying them one at a time across a chat gets to drop or transpose one. They
// travel together or they are wrong together — which is what the packed form
// makes true mechanically instead of by instruction.
//
// It is an envelope, not a protection. The secret inside is plain: base64 is a
// transport convenience, and the block still has to go over a channel the
// operator would trust with a password.
package federationkey

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Version is stamped into every key. A reader that meets a version it does not
// know says so instead of guessing at fields it has never seen — the alternative
// is a half-decoded pairing that fails later and further away.
const Version = 1

// ErrUnsupportedVersion means the key was written by a newer trip2g.
var ErrUnsupportedVersion = errors.New("federation key version is not supported")

// Handover is what the side that owns the knowledge tells the side that wants it.
//
// KBID is a suggestion, not an instruction: it is the slug the asking side will
// address this base by, and only that side's vault records it. It rides along
// because the person doing the wiring needs it for the KB-note and would
// otherwise have to be told separately.
type Handover struct {
	Version   int    `json:"v"`
	KID       string `json:"kid"`
	KBURL     string `json:"kb_url"`
	SecretHex string `json:"secret_hex"`
	KBID      string `json:"kb_id,omitempty"`
}

// Encode packs a handover into the string an operator copies.
func Encode(handover Handover) (string, error) {
	handover.Version = Version

	raw, err := json.Marshal(handover)
	if err != nil {
		return "", fmt.Errorf("marshal federation key: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Decode unpacks one, tolerating the whitespace a value picks up on its way
// through a chat window and a clipboard.
func Decode(key string) (Handover, error) {
	trimmed := strings.Join(strings.Fields(key), "")
	if trimmed == "" {
		return Handover{}, errors.New("federation key is empty")
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(trimmed, "="))
	if err != nil {
		return Handover{}, fmt.Errorf("federation key is not a valid key: %w", err)
	}

	var handover Handover
	err = json.Unmarshal(raw, &handover)
	if err != nil {
		return Handover{}, fmt.Errorf("federation key is not a valid key: %w", err)
	}

	if handover.Version != Version {
		return Handover{}, fmt.Errorf("%w: %d", ErrUnsupportedVersion, handover.Version)
	}
	if handover.KID == "" || handover.KBURL == "" || handover.SecretHex == "" {
		return Handover{}, errors.New("federation key is missing the kid, the address or the secret")
	}

	return handover, nil
}
