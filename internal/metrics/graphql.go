package metrics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/prometheus/client_golang/prometheus"
)

// GraphQLMetrics collects Prometheus metrics for GraphQL operations and resolvers.
// Implements graphql.HandlerExtension and graphql.FieldInterceptor for srv.Use().
type GraphQLMetrics struct {
	// Operation-level (per query name).
	opDuration *prometheus.HistogramVec
	opRequests *prometheus.CounterVec

	// Resolver-level (per field).
	resolverDuration *prometheus.HistogramVec
	resolverStarted  *prometheus.CounterVec
	resolverDone     *prometheus.CounterVec
}

var _ interface {
	graphql.HandlerExtension
	graphql.FieldInterceptor
} = (*GraphQLMetrics)(nil)

// NewGraphQLMetrics creates and registers all GraphQL Prometheus metrics.
func NewGraphQLMetrics() *GraphQLMetrics {
	// ExponentialBuckets starting at 0.001s (1ms), x2, 11 buckets → up to ~1s.
	resolverBuckets := prometheus.ExponentialBuckets(0.001, 2, 11)
	opBuckets := prometheus.DefBuckets

	opDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "trip2g_graphql_duration_seconds",
		Help:    "GraphQL operation duration in seconds",
		Buckets: opBuckets,
	}, []string{"type", "name"})
	prometheus.MustRegister(opDuration)

	opRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trip2g_graphql_requests_total",
		Help: "Total number of GraphQL requests",
	}, []string{"type", "name", "status"})
	prometheus.MustRegister(opRequests)

	resolverDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "trip2g_graphql_resolver_duration_seconds",
		Help:    "GraphQL resolver duration in seconds",
		Buckets: resolverBuckets,
	}, []string{"status", "object", "field"})
	prometheus.MustRegister(resolverDuration)

	resolverStarted := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trip2g_graphql_resolver_started_total",
		Help: "Total number of GraphQL resolver calls started",
	}, []string{"object", "field"})
	prometheus.MustRegister(resolverStarted)

	resolverDone := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trip2g_graphql_resolver_done_total",
		Help: "Total number of GraphQL resolver calls completed",
	}, []string{"object", "field"})
	prometheus.MustRegister(resolverDone)

	return &GraphQLMetrics{
		opDuration:       opDuration,
		opRequests:       opRequests,
		resolverDuration: resolverDuration,
		resolverStarted:  resolverStarted,
		resolverDone:     resolverDone,
	}
}

// ExtensionName implements graphql.HandlerExtension.
func (m *GraphQLMetrics) ExtensionName() string {
	return "PrometheusMetrics"
}

// Validate implements graphql.HandlerExtension.
func (m *GraphQLMetrics) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

// InterceptField implements graphql.FieldInterceptor — resolver-level metrics.
func (m *GraphQLMetrics) InterceptField(ctx context.Context, next graphql.Resolver) (any, error) {
	fc := graphql.GetFieldContext(ctx)
	m.resolverStarted.WithLabelValues(fc.Object, fc.Field.Name).Inc()

	start := time.Now()
	res, err := next(ctx)
	elapsed := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "error"
	}

	m.resolverDuration.WithLabelValues(status, fc.Object, fc.Field.Name).Observe(elapsed)
	m.resolverDone.WithLabelValues(fc.Object, fc.Field.Name).Inc()

	return res, err
}

// Middleware returns a gqlgen AroundOperations middleware — operation-level metrics.
// Label format: "OperationName_sha256prefix" or "anon_sha256prefix".
func (m *GraphQLMetrics) Middleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		opCtx := graphql.GetOperationContext(ctx)
		op := opCtx.Operation

		opType := string(op.Operation)
		opName := labelName(op.Name, opCtx.RawQuery)

		start := time.Now()
		rh := next(ctx)

		return func(ctx context.Context) *graphql.Response {
			resp := rh(ctx)
			if resp == nil {
				return nil
			}

			status := "success"
			if len(resp.Errors) > 0 {
				status = "error"
			}

			m.opDuration.WithLabelValues(opType, opName).Observe(time.Since(start).Seconds())
			m.opRequests.WithLabelValues(opType, opName, status).Add(1)

			return resp
		}
	}
}

// labelName builds the prometheus label: "Name_sha256prefix" or "anon_sha256prefix".
func labelName(name, rawQuery string) string {
	hash := sha256.Sum256([]byte(rawQuery))
	prefix := hex.EncodeToString(hash[:4])

	if name == "" {
		return "anon_" + prefix
	}

	return name + "_" + prefix
}
