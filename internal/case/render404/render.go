package render404

import (
	"net/http"
	"trip2g/internal/appreq"
)

func Handle(req *appreq.Request) (interface{}, error) {
	// TODO: extract 404 from other handlers
	ctx := req.Req
	ctx.SetStatusCode(http.StatusNotFound)

	return nil, nil
}
