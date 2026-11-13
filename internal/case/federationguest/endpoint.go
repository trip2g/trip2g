package federationguest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"trip2g/internal/appreq"
	"trip2g/internal/appresp"
)

const (
	Path        = "/_system/federation/guest"
	TicketParam = "ticket"

	msgRefused = "federation_guest_refused"
)

// Endpoint takes the handoff the hub's page posts and answers with the ticket
// for the GET leg.
type Endpoint struct{}

func (*Endpoint) Path() string   { return Path }
func (*Endpoint) Method() string { return http.MethodPost }

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	form := Form{
		Token:  string(req.Req.PostArgs().Peek("token")),
		Path:   string(req.Req.PostArgs().Peek("path")),
		Origin: string(req.Req.Request.Header.Peek("Origin")),
	}

	ticket, err := Admit(req.Req, req.Env.(Env), form)
	if err != nil {
		return nil, failure(err)
	}

	query := url.Values{TicketParam: {ticket}}
	appresp.Redirect(req.Req, Path+"?"+query.Encode(), http.StatusSeeOther)
	return nil, nil
}

// GetEndpoint exchanges a ticket for a guest session, unless the browser
// already holds a session of its own here: that one wins and is left alone.
type GetEndpoint struct{}

func (*GetEndpoint) Path() string   { return Path }
func (*GetEndpoint) Method() string { return http.MethodGet }

func (*GetEndpoint) Handle(req *appreq.Request) (interface{}, error) {
	data, path, err := Enter(req.Env.(Env), string(req.Req.QueryArgs().Peek(TicketParam)))
	if err != nil {
		return nil, failure(err)
	}

	current, err := req.UserToken()
	if err != nil {
		return nil, fmt.Errorf("read current session: %w", err)
	}

	if current == nil || current.IsFederationGuest() {
		stored, storeErr := req.TokenManager.Store(req.Req, data)
		if storeErr != nil {
			return nil, fmt.Errorf("store guest session: %w", storeErr)
		}
		req.SetUserToken(&stored.Data)
	}

	appresp.Redirect(req.Req, path, http.StatusFound)
	return nil, nil
}

func failure(err error) error {
	code := http.StatusInternalServerError
	if errors.Is(err, ErrRefused) {
		code = http.StatusForbidden
	}
	return &appreq.SystemMessageError{Code: code, Msg: msgRefused, Err: err}
}
