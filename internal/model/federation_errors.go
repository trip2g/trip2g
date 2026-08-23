package model

import "fmt"

// FederationRPCError is a JSON-RPC error response from a peer. It is an answer
// — the peer heard the call and named a reason — where a transport failure
// proves nothing reached it. Callers that record opposite outcomes for the two
// (key rotation does) branch on the code; everyone else treats it as any other
// error. Kept out of federation.go so easyjson does not generate for it.
type FederationRPCError struct {
	Code    int
	Message string
}

func (e *FederationRPCError) Error() string {
	return fmt.Sprintf("federation rpc error %d: %s", e.Code, e.Message)
}

// FederationHTTPError is a non-2xx HTTP answer from a peer's endpoint. Like
// FederationRPCError it proves something answered; unlike it, the answer came
// from the HTTP layer — a proxy, an adapter without the endpoint — before any
// JSON-RPC was dispatched.
type FederationHTTPError struct {
	Status int
}

func (e *FederationHTTPError) Error() string {
	return fmt.Sprintf("post json: status %d", e.Status)
}
