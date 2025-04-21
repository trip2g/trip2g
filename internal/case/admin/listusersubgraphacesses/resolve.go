package listusersubgraphacesses

import (
	"context"
	"trip2g/internal/appresp"
	"trip2g/internal/db"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	ListUserSubgraphAccesses(ctx context.Context) ([]db.ListUserSubgraphAccessesRow, error)
	ListAdminSubgraphs(ctx context.Context) ([]db.Subgraph, error)
	ListUsersByIDs(ctx context.Context, ids []int64) ([]db.User, error)
}

type Request struct{}

func (r *Request) Validate() error {
	return nil
}

type Response struct {
	appresp.Response
	Rows      []db.ListUserSubgraphAccessesRow
	Subgraphs []db.Subgraph
	Users     []db.User
}

func Resolve(ctx context.Context, env Env, request Request) (Response, error) {
	var response Response
	response.Success = true

	accesses, err := env.ListUserSubgraphAccesses(ctx)
	if err != nil {
		return response, err
	}

	userIDsMap := make(map[int64]struct{})
	userIDs := make([]int64, 0, len(accesses))

	for _, access := range accesses {
		_, ok := userIDsMap[access.UserID]
		if !ok {
			userIDsMap[access.UserID] = struct{}{}
			userIDs = append(userIDs, access.UserID)
		}
	}

	users, err := env.ListUsersByIDs(ctx, userIDs)
	if err != nil {
		return response, err
	}

	subgraphs, err := env.ListAdminSubgraphs(ctx)
	if err != nil {
		return response, err
	}

	response.Rows = accesses
	response.Subgraphs = subgraphs
	response.Users = users

	return response, nil
}
