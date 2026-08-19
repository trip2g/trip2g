package router

import (
	"errors"
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/defaulttemplate"
	"unsafe"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
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
		if writeSystemMessage(ctx, err) {
			// The request was answered, so the error must not travel up:
			// cmd/server would replace the page with a 503. The cause is worth
			// keeping, though — it is no longer in the response body.
			router.env.Logger().Error("answered with a system message", "err", err, "path", path)
			return true, nil
		}

		jsonErr, isJSONErr := err.(easyjson.Marshaler)
		if isJSONErr {
			ctx.SetStatusCode(http.StatusBadRequest)
			ctx.SetContentType("application/json")

			rawBytes, marshalErr := easyjson.Marshal(jsonErr)
			if marshalErr != nil {
				router.env.Logger().Error("failed to marshal error response", "err", marshalErr, "path", path)
				ctx.SetBody([]byte(marshalErr.Error()))
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

// writeSystemMessage renders the plain "here is what happened" page for an
// endpoint that asked for one, and reports whether it did. Endpoints whose
// caller is a person in a browser return appreq.SystemMessageError instead of
// letting a wrapped internal error become the response body.
func writeSystemMessage(ctx *fasthttp.RequestCtx, err error) bool {
	var sysErr *appreq.SystemMessageError
	if !errors.As(err, &sysErr) {
		return false
	}

	defaulttemplate.WriteSystemMessage(ctx, sysErr.Code, sysErr.Msg)

	return true
}

// read https://github.com/valyala/fasthttp?tab=readme-ov-file#tricks-with-byte-buffers.
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
