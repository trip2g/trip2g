package processnowpaymentsipn

import (
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/nowpayments"

	easyjson "github.com/mailru/easyjson"
)

type Endpoint struct{}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	request := nowpayments.IPNRequest{}
	env := req.Env.(Env)
	sig := string(req.Req.Request.Header.Peek("x-nowpayments-sig"))

	ok, err := nowpayments.CheckIPNSignature(env.NowpaymentsIPNSecret(), sig, req.Req.PostBody())
	if err != nil {
		env.Logger().Error("failed to CheckIPNSignature", "error", err)
	}

	if !ok || err != nil {
		env.Logger().Error("invalid IPN signature", "error", err, "sig", sig)
		req.Req.SetStatusCode(http.StatusNotFound)
		return nil, nil
	}

	env.Logger().Info("IPN request", "body", string(req.Req.PostBody()), "sig", sig)

	err = easyjson.Unmarshal(req.Req.PostBody(), &request)
	if err != nil {
		return nil, err
	}

	return Resolve(req.Req, req.Env.(Env), request)
}

func (*Endpoint) Path() string {
	return "/api/ipn/nowpayments"
}

func (*Endpoint) Method() string {
	return http.MethodPost
}
