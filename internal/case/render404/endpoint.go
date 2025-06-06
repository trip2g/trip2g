package render404

import (
	"net/http"
	"trip2g/internal/appreq"
)

// Endpoint ensures that the Env interface is implemented for the router.
type Endpoint struct{}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	return Handle(req)
}

func (Endpoint) Path() string {
	return "/_/system/404"
}

func (Endpoint) Method() string {
	return http.MethodGet
}
