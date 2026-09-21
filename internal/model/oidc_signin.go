package model

// OIDCSignIn describes the OIDC sign-in entry point: where the provider sends
// the user back, where the button points, and what the button says. Label is
// empty unless the provider was given a display name, and the frontend then
// falls back to its own wording.
type OIDCSignIn struct {
	CallbackURL string
	AuthURL     string
	Label       string
}
