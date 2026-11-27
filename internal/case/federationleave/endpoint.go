// Package federationleave ends a federation guest's session from the bar on
// every page, and sends the guest back to the hub they came from.
package federationleave

import (
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/appresp"
	"trip2g/internal/case/federationguest"
)

const (
	Path = "/_system/federation/leave"

	msgRefused = "federation_leave_refused"
)

type Env interface {
	Insecure() bool
}

// Endpoint ends the session of a guest and nobody else's: a reader with a
// session of their own is not a guest, and keeps it.
type Endpoint struct{}

func (*Endpoint) Path() string   { return Path }
func (*Endpoint) Method() string { return http.MethodPost }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	token, err := req.UserToken()
	if err != nil {
		return nil, err
	}
	if !token.IsFederationGuest() {
		return nil, &appreq.SystemMessageError{Code: http.StatusForbidden, Msg: msgRefused}
	}

	err = req.TokenManager.Delete(req.Req)
	if err != nil {
		return nil, err
	}
	federationguest.ForgetWasGuest(req.Req, req.Env.(Env).Insecure())

	back := "/"
	if token.Origin != "" {
		back = token.Origin + "/"
	}
	appresp.Redirect(req.Req, back, http.StatusSeeOther)
	return nil, nil
}
