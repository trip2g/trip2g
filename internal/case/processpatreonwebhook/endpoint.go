package processpatreonwebhook

import (
	"context"
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

	// Get webhook signature from header
	signature := string(req.Req.Request.Header.Peek("X-Patreon-Signature"))
	if signature == "" {
		env.Logger().Error("missing X-Patreon-Signature header", "credential_id", credentialID)
		req.Req.SetStatusCode(http.StatusBadRequest)
		return nil, nil
	}

	// Create request struct
	webhookRequest := Request{
		CredentialID: credentialID,
		Signature:    signature,
		Body:         req.Req.PostBody(),
	}

	// Log webhook request
	env.Logger().Info("Patreon webhook request received",
		"credential_id", credentialID,
		"event", string(req.Req.Request.Header.Peek("X-Patreon-Event")),
	)

	ctx := context.Background()
	response, err := Resolve(ctx, env, webhookRequest)
	if err != nil {
		env.Logger().Error("webhook processing failed", "credential_id", credentialID, "error", err)
		req.Req.SetStatusCode(http.StatusBadRequest)
		return nil, nil
	}

	return response, nil
}

func (*Endpoint) Path() string {
	return "/api/patreon/webhook"
}

func (*Endpoint) Method() string {
	return http.MethodPost
}
