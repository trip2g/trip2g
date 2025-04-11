package router

import (
	"errors"
	"net/http"
	"trip2g/internal/appreq"

	"github.com/mailru/easyjson"
)

type Endpoint interface {
	Handle(req *appreq.Request) (interface{}, error)
	Path() string
	Method() string
}

type Router struct {
	env Env

	prefixLen  int
	getRoutes  map[string]Endpoint
	postRoutes map[string]Endpoint
}

var ErrNotFound = errors.New("not found")

func New(env Env, prefix string) *Router {
	router := Router{
		env: env,

		prefixLen:  len(prefix),
		getRoutes:  make(map[string]Endpoint),
		postRoutes: make(map[string]Endpoint),
	}

	for _, endpoint := range endpoints {
		switch endpoint.Method() {
		case http.MethodGet:
			router.getRoutes[endpoint.Path()] = endpoint
		case http.MethodPost:
			router.postRoutes[endpoint.Path()] = endpoint
		default:
			panic("unsupported method")
		}
	}

	return &router
}

// Handle returns true if the request was handled.
func (router *Router) Handle(req *appreq.Request) bool {
	var routes map[string]Endpoint

	ctx := req.Req

	switch string(ctx.Method()) {
	case http.MethodGet:
		routes = router.getRoutes
	case http.MethodPost:
		routes = router.postRoutes
	default:
		return false
	}

	path := string(ctx.Path()[router.prefixLen:])

	endpoint, ok := routes[path]
	if !ok {
		return false
	}

	respI, err := endpoint.Handle(req)
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return true
	}

	resp, ok := respI.(easyjson.Marshaler)
	if !ok {
		panic("response is not easyjson.Marshaler")
	}

	rawBytes, err := easyjson.Marshal(resp)
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return true
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("application/json")
	ctx.SetBody(rawBytes)

	return true
}
