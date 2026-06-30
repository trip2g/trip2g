package fleet

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"

	"trip2g/internal/fleet/trip2ggql"
)

// gqlDoerFunc adapts a func to genqlient's graphql.Doer for tests.
type gqlDoerFunc func(*http.Request) (*http.Response, error)

func (f gqlDoerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

// emptyCronNodesData is the ListCronWebhooks response for an empty cron-webhook set.
const emptyCronNodesData = `{"admin":{"allCronWebhooks":{"nodes":[]}}}`

// fakeAdminGQL builds a genqlient client whose transport routes each admin-lane
// request to respond, keyed by GraphQL operationName. respond returns the JSON
// object placed under the {"data": ...} envelope; a non-nil error is sent as a
// GraphQL errors[] entry. The real genqlient decode path runs over the result,
// so a field/typename mismatch surfaces in the test.
//
// ListCronWebhooks is automatically handled with an empty-list response when
// the respond func doesn't cover it (returns an error). This lets existing
// change-mode tests remain unchanged after cron reconcile was added.
func fakeAdminGQL(respond func(op string, vars json.RawMessage) (string, error)) graphql.Client {
	innerRespond := respond
	respond = func(op string, vars json.RawMessage) (string, error) {
		data, err := innerRespond(op, vars)
		if err != nil && op == "ListCronWebhooks" {
			return emptyCronNodesData, nil
		}
		return data, err
	}
	doer := gqlDoerFunc(func(req *http.Request) (*http.Response, error) {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		var in struct {
			OperationName string          `json:"operationName"`
			Variables     json.RawMessage `json:"variables"`
		}
		if uerr := json.Unmarshal(raw, &in); uerr != nil {
			return nil, uerr
		}
		data, rerr := respond(in.OperationName, in.Variables)
		var env string
		if rerr != nil {
			env = `{"errors":[{"message":` + strconv.Quote(rerr.Error()) + `}]}`
		} else {
			env = `{"data":` + data + `}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(env)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
	return graphql.NewClient("http://fake.local/_system/graphql", doer)
}

// createInputFrom decodes a CreateChangeWebhook request's typed input variable.
func createInputFrom(t *testing.T, vars json.RawMessage) trip2ggql.ChangeWebhookCreateInput {
	t.Helper()
	var v struct {
		Input trip2ggql.ChangeWebhookCreateInput `json:"input"`
	}
	require.NoError(t, json.Unmarshal(vars, &v))
	return v.Input
}

// deleteIDFrom decodes a DeleteChangeWebhook request's id variable.
func deleteIDFrom(t *testing.T, vars json.RawMessage) int64 {
	t.Helper()
	var v struct {
		Input trip2ggql.ChangeWebhookDeleteInput `json:"input"`
	}
	require.NoError(t, json.Unmarshal(vars, &v))
	return v.Input.Id
}
