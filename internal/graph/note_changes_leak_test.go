package graph

// Regression tests for the NoteChanges subscription goroutine leak.
//
// Production incident: ~820 NoteChanges subscription goroutines leaked (idle up
// to 748 min) because a disconnected SSE client never cancelled the subscription
// context. The resolver goroutine itself is correct — it exits and unsubscribes
// from notebus on ctx.Done — but over fasthttp a client disconnect is only
// observable through a failed write, and transport.SSE was registered without
// KeepAlivePingInterval, so an idle stream never wrote anything and never
// learned the client was gone.
//
// These tests pin down both halves of the contract:
//   - sseTransport() must have keepalive enabled (the actual fix);
//   - the resolver goroutine must exit and unsubscribe from notebus once its
//     ctx is cancelled, including while parked on a payload send.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
	"trip2g/internal/notebus"
	"trip2g/internal/usertoken"
)

// TestSSETransport_KeepAliveEnabled guards the root-cause fix: without a
// keepalive ping the fasthttp SSE bridge (internal/fastgql) has no write on an
// idle stream, so a dead client is never detected and the subscription ctx is
// never cancelled — leaking the resolver goroutine and its notebus subscriber.
func TestSSETransport_KeepAliveEnabled(t *testing.T) {
	require.Positive(t, sseTransport().KeepAlivePingInterval,
		"SSE transport must send keepalive pings: over fasthttp a client disconnect "+
			"is only observable through a failed write, so an idle noteChanges stream "+
			"would otherwise leak its goroutine and notebus subscriber forever")
}

// noteChangesLeakEnv implements just the Env surface NoteChanges touches, over
// a real notebus.Bus. The embedded nil Env panics on any unexpected call.
type noteChangesLeakEnv struct {
	Env
	bus *notebus.Bus
	nvs *appmodel.NoteViews
}

func (e *noteChangesLeakEnv) SubscribeNoteChanges(include, exclude []string) *notebus.Subscriber {
	return e.bus.Subscribe(include, exclude, 1)
}

func (e *noteChangesLeakEnv) UnsubscribeNoteChanges(sub *notebus.Subscriber) {
	e.bus.Unsubscribe(sub)
}

func (e *noteChangesLeakEnv) CurrentUserToken(_ context.Context) (*usertoken.Data, error) {
	return &usertoken.Data{ID: 1, Role: "admin"}, nil
}

//nolint:staticcheck // name fixed by the checkapikey.Env interface
func (e *noteChangesLeakEnv) ApiKeyByValue(_ context.Context, _ string) (db.ApiKey, error) {
	return db.ApiKey{}, sql.ErrNoRows
}

func (e *noteChangesLeakEnv) InsertAPIKeyLog(_ context.Context, _ db.InsertAPIKeyLogParams) error {
	return nil
}

func (e *noteChangesLeakEnv) UpsertAPIKeyLogAction(_ context.Context, _ string) error { return nil }

func (e *noteChangesLeakEnv) UpsertAPIKeyLogIP(_ context.Context, _ string) error { return nil }

func (e *noteChangesLeakEnv) ShortAPITokenSecret() string { return testShortAPISecret }

func (e *noteChangesLeakEnv) LatestNoteViews() *appmodel.NoteViews { return e.nvs }

func (e *noteChangesLeakEnv) CanReadNote(_ context.Context, _ *appmodel.NoteView) (bool, error) {
	return true, nil
}

func (e *noteChangesLeakEnv) Logger() logger.Logger { return &logger.DummyLogger{} }

// cancelableAuthCtx combines a cancellable base context with an authCtx-built
// fasthttp context for Value lookups (a bare fasthttp.RequestCtx cannot be a
// context.WithCancel parent — its Done() needs a running server).
type cancelableAuthCtx struct {
	context.Context
	values context.Context
}

func (c *cancelableAuthCtx) Value(key any) any { return c.values.Value(key) }

func startLeakSubscription(t *testing.T) (*notebus.Bus, <-chan *model.NoteChangesSubscriptionPayload, context.CancelFunc) {
	t.Helper()

	bus := notebus.New(&logger.DummyLogger{})
	env := &noteChangesLeakEnv{
		bus: bus,
		nvs: aclNvs(aclNote(1, "a/one.md", "/a-one", "a")),
	}
	r := &subscriptionResolver{&Resolver{DefaultEnv: env}}

	base, cancel := context.WithCancel(context.Background())
	ctx := &cancelableAuthCtx{Context: base, values: authCtx(&usertoken.Data{ID: 1, Role: "admin"}, nil)}

	ch, err := r.NoteChanges(ctx, model.NoteChangesFilter{IncludePatterns: []string{"**"}})
	require.NoError(t, err)
	require.Equal(t, 1, bus.Stats().Subscribers, "subscription must register on notebus")

	return bus, ch, cancel
}

// requireSubscriptionGone asserts the resolver goroutine exited (payload
// channel closed) and the notebus subscriber count returned to baseline.
func requireSubscriptionGone(t *testing.T, bus *notebus.Bus, ch <-chan *model.NoteChangesSubscriptionPayload) {
	t.Helper()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				require.Eventually(t, func() bool { return bus.Stats().Subscribers == 0 },
					5*time.Second, 10*time.Millisecond,
					"notebus subscriber must be removed after the goroutine exits")
				return
			}
			// drain payloads buffered before cancellation
		case <-deadline:
			t.Fatal("subscription goroutine did not exit after ctx cancel")
		}
	}
}

func TestNoteChanges_CtxCancelWhileIdle_ExitsAndUnsubscribes(t *testing.T) {
	bus, ch, cancel := startLeakSubscription(t)

	cancel()

	requireSubscriptionGone(t, bus, ch)
}

func TestNoteChanges_CtxCancelWhileSendBlocked_ExitsAndUnsubscribes(t *testing.T) {
	bus, ch, cancel := startLeakSubscription(t)

	// Nobody reads ch (cap 1): the first publish fills the payload buffer, the
	// second parks the goroutine on the payload send. Cancel must unblock it.
	batch := notebus.Batch{Changes: []notebus.Change{{PathID: 1, Path: "a/one.md", Event: "remove"}}}
	bus.Publish(batch)
	require.Eventually(t, func() bool { return len(ch) == 1 }, 5*time.Second, 10*time.Millisecond,
		"first payload must be buffered")
	bus.Publish(batch)
	time.Sleep(50 * time.Millisecond) // let the goroutine park on the blocked send

	cancel()

	requireSubscriptionGone(t, bus, ch)
}
