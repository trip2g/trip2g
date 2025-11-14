package checkhealth

import (
	"context"
	"fmt"

	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	GetPublicURLForRequest(ctx context.Context) string
}

// Checker interface for health check implementations.
type Checker interface {
	ID() string
	Check(ctx context.Context, env Env) model.HealchCheck
}

// getCheckers returns the list of all health checkers.
func getCheckers() []Checker {
	return []Checker{
		&GraphQLIntrospectionChecker{},
		&AdminAuthorizationChecker{},
		&APIKeyValidationChecker{},
	}
}

func Resolve(ctx context.Context, env Env) ([]model.HealchCheck, error) {
	token, err := env.CurrentAdminUserToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user token: %w", err)
	}
	_ = token // token validated, unused for now but available if needed

	checkers := getCheckers()
	results := make([]model.HealchCheck, 0, len(checkers))

	for _, checker := range checkers {
		result := checker.Check(ctx, env)
		results = append(results, result)
	}

	return results, nil
}
