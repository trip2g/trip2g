package graph

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/vektah/gqlparser/v2/ast"
)

func NewHandler(env Env) *handler.Server {
	log := env.Logger()

	resolver := Resolver{DefaultEnv: env}
	srv := handler.New(NewExecutableSchema(Config{Resolvers: &resolver}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: 30 * 1024 * 1024,
		MaxMemory:     10 * 1024 * 1024,
	})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	graphqlErr := func(err error) graphql.ResponseHandler {
		log.Error("graphql error", "error", err)

		return func(ctx context.Context) *graphql.Response {
			return graphql.ErrorResponse(ctx, "%s", err.Error())
		}
	}

	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		operationContext := graphql.GetOperationContext(ctx)

		if operationContext.Operation.Operation == ast.Mutation {
			err := env.AcquireTxEnvInRequest(ctx, operationContext.Operation.Name)
			if err != nil {
				log.Error("failed to acquire transactioned env", "error", err)
				return graphqlErr(err)
			}

			rh := next(ctx)

			return func(ctx context.Context) *graphql.Response {
				resp := rh(ctx)

				// А тут интересно, нужно ли отказывать транзакции только в случае ошибок
				// или в случае ErrorPayload так же нужно? Похоже нужно откатывать в случае
				// непредвиденных ошибок и дополнительно вводить специальную ошибку для rollback.
				if len(resp.Errors) > 0 {
					err := env.ReleaseTxEnvInRequest(ctx, false)
					if err != nil {
						log.Error("failed to release transactioned env with rollback", "error", err)
					} else {
						log.Info("released transactioned env with rollback")
					}
				} else {
					err := env.ReleaseTxEnvInRequest(ctx, true)
					if err != nil {
						log.Error("failed to release transactioned env with commit", "error", err)
					} else {
						log.Debug("released transactioned env with commit")
					}
				}

				return resp
			}
		}

		return next(ctx)
	})

	return srv
}
