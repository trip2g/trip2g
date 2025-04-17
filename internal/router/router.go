package router

import (
	"errors"
	"net/http"
	"trip2g/internal/appreq"
	"unsafe"

	"github.com/mailru/easyjson"
)

//go:generate go run ./gencmd

type Endpoint interface {
	Handle(req *appreq.Request) (interface{}, error)
	Path() string
	Method() string
}

type Router struct {
	env Env

	getRoutes  map[string]Endpoint
	postRoutes map[string]Endpoint

	notFoundEndpoint Endpoint
}

var ErrNotFound = errors.New("not found")

func New(env Env) *Router {
	router := Router{
		env: env,

		getRoutes:  make(map[string]Endpoint),
		postRoutes: make(map[string]Endpoint),
	}

	for _, endpoint := range endpoints {
		path := endpoint.Path()
		if path == "" {
			if router.notFoundEndpoint != nil {
				panic("duplicate not found endpoint")
			}

			router.notFoundEndpoint = endpoint

			env.Logger().Info("found not found endpoint")
		}

		switch endpoint.Method() {
		case http.MethodGet:
			_, ok := router.getRoutes[path]
			if ok {
				panic("duplicate endpoint: " + path)
			}

			router.getRoutes[path] = endpoint

		case http.MethodPost:
			_, ok := router.postRoutes[path]
			if ok {
				panic("duplicate endpoint: " + path)
			}

			router.postRoutes[path] = endpoint

		default:
			panic("unsupported method")
		}
	}

	return &router
}

// Handle returns true if the request was handled.
func (router *Router) Handle(req *appreq.Request) (bool, error) {
	rawPath := req.Req.URI().Path()
	path := b2s(rawPath)
	method := b2s(req.Req.Method())
	ctx := req.Req

	var endpoint Endpoint
	var ok bool

	switch method {
	case http.MethodGet:
		endpoint, ok = router.getRoutes[path]
	case http.MethodPost:
		endpoint, ok = router.postRoutes[path]
	}

	if !ok {
		if router.notFoundEndpoint != nil {
			endpoint = router.notFoundEndpoint
		} else {
			return false, nil
		}
	}

	respI, err := endpoint.Handle(req)
	if err != nil {
		jsonErr, ok := err.(easyjson.Marshaler)
		if ok {
			ctx.SetStatusCode(http.StatusBadRequest)
			ctx.SetContentType("application/json")

			rawBytes, err := easyjson.Marshal(jsonErr)
			if err != nil {
				router.env.Logger().Error("failed to marshal error response", "err", err, "path", path)
				ctx.SetBody([]byte(err.Error()))
				return true, err
			}

			ctx.SetBody(rawBytes)
			return true, nil
		}

		router.env.Logger().Error("failed to handle request", "err", err, "path", path)
		ctx.SetStatusCode(http.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return true, err
	}

	resp, ok := respI.(easyjson.Marshaler)
	if !ok {
		// the handler must write the response itself
		return true, nil
	}

	rawBytes, err := easyjson.Marshal(resp)
	if err != nil {
		router.env.Logger().Error("failed to marshal response", "err", err, "path", path)
		ctx.SetStatusCode(http.StatusInternalServerError)
		ctx.SetBody([]byte(err.Error()))
		return true, err
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.SetContentType("application/json")
	ctx.SetBody(rawBytes)

	return true, nil
}

// read https://github.com/valyala/fasthttp?tab=readme-ov-file#tricks-with-byte-buffers.
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
