package router

import (
	"errors"
	"net/http"
	"trip2g/internal/appreq"
	"unsafe"

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

	rawPath := ctx.Path()
	if len(rawPath) <= router.prefixLen {
		return false
	}

	if b2s(ctx.Method()) == http.MethodGet {
		routes = router.getRoutes
	} else {
		routes = router.postRoutes
	}

	path := b2s(rawPath[router.prefixLen:])

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
		// the handler must write the response itself
		return true
	}

	rawBytes, err := easyjson.Marshal(resp)
	if err != nil {
		router.env.Logger().Error("failed to marshal response", "err", err, "path", path)
		ctx.SetStatusCode(http.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return true
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("application/json")
	ctx.SetBody(rawBytes)

	return true
}

// read https://github.com/valyala/fasthttp?tab=readme-ov-file#tricks-with-byte-buffers.
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
