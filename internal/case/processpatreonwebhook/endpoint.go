package processpatreonwebhook

import (
	"net/http"
	"strconv"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)

	// Get credential_id from query parameter
	credentialIDStr := string(req.Req.Request.URI().QueryArgs().Peek("credential_id"))
	if credentialIDStr == "" {
		env.Logger().Error("missing credential_id parameter")
		req.Req.SetStatusCode(http.StatusBadRequest)
		return nil, nil
	}

	// Parse credential ID
	credentialID, err := strconv.ParseInt(credentialIDStr, 10, 64)
	if err != nil {
		env.Logger().Error("invalid credential_id", "credential_id", credentialIDStr, "error", err)
		req.Req.SetStatusCode(http.StatusBadRequest)
		return nil, nil
	}

	// Log webhook request
	env.Logger().Info("Patreon webhook request",
		"credential_id", credentialID,
		"body", string(req.Req.PostBody()),
	)

	return Resolve(req.Req, req.Env.(Env), credentialID)
}

func (*Endpoint) Path() string {
	return "/api/patreon/webhook"
}

func (*Endpoint) Method() string {
	return http.MethodPost
}
