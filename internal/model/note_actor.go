package model

// NoteActor identifies who pushed a note version: the signed-in user,
// the authenticating API key, and the client identifier from the
// X-trip2g-client request header. Any field may be nil/empty when unknown.
type NoteActor struct {
	UserID   *int64
	APIKeyID *int64
	Client   *string
}
