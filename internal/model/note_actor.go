package model

// NoteActor identifies who pushed a note version: the signed-in user,
// the authenticating API key, the client identifier from the
// X-trip2g-client request header, and the fleet delivery that produced it.
// Any field may be nil/empty when unknown.
type NoteActor struct {
	UserID       *int64
	APIKeyID     *int64
	Client       *string
	DeliveryKind *string
	DeliveryID   *int64
}
