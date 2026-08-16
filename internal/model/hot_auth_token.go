package model

type HotAuthToken struct {
	Email string `json:"e"`
	// Provisioning: creates the user if missing and makes them admin. Only for
	// fleet logic — the fleet and the login-link CLI mint it in-process from the
	// JWT secret, to reach an instance nobody can sign in to yet. The admin API
	// cannot ask for it.
	AdminEnter bool `json:"ae,omitempty"`
	// Where the exchange sends the browser. Empty keeps each endpoint's own
	// default.
	Redirect string `json:"r,omitempty"`
}
