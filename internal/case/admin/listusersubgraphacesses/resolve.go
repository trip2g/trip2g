package listusersubgraphacesses

import (
	"context"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	ListUserSubgraphAccesses(ctx context.Context) ([]db.ListUserSubgraphAccessesRow, error)
}

type Request struct{}

func (r *Request) Validate() error {
	return nil
}

type Response struct {
	appresp.Response
	Accesses []db.ListUserSubgraphAccessesRow
}

func Resolve(ctx context.Context, env Env, request Request) (Response, error) {
	var response Response
	response.Success = true

	accesses, err := env.ListUserSubgraphAccesses(ctx)
	if err != nil {
		return response, err
	}

	response.Accesses = accesses
	return response, nil
}
