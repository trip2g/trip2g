package jobs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

type registerTestParams struct {
	Value string `json:"value"`
}

type registerTestEnv struct {
	registeredID     string
	registeredQueue  model.BackgroundQueueID
	handler          func(ctx context.Context, m []byte) error
	enqueuedTask     model.BackgroundTask
	enqueueCallCount int
}

func (e *registerTestEnv) RegisterJob(qID model.BackgroundQueueID, id string, handler func(ctx context.Context, m []byte) error) {
	e.registeredQueue = qID
	e.registeredID = id
	e.handler = handler
}

func (e *registerTestEnv) EnqueueJob(_ context.Context, job model.BackgroundTask) error {
	e.enqueueCallCount++
	e.enqueuedTask = job
	return nil
}

func TestRegisterHandlerUnmarshalsAndCallsThrough(t *testing.T) {
	env := &registerTestEnv{}
	var calledCtx context.Context
	var calledParams registerTestParams

	Register(env, model.BackgroundDefaultQueue, "test_job", 5,
		func(ctx context.Context, params registerTestParams) error {
			calledCtx = ctx
			calledParams = params
			return nil
		})

	require.Equal(t, model.BackgroundDefaultQueue, env.registeredQueue)
	require.Equal(t, "test_job", env.registeredID)
	require.NotNil(t, env.handler)

	ctx := context.Background()
	err := env.handler(ctx, []byte(`{"value":"hello"}`))

	require.NoError(t, err)
	require.Equal(t, ctx, calledCtx)
	require.Equal(t, registerTestParams{Value: "hello"}, calledParams)
}

func TestRegisterHandlerWrapsUnmarshalError(t *testing.T) {
	env := &registerTestEnv{}

	Register(env, model.BackgroundDefaultQueue, "test_job", 5,
		func(_ context.Context, _ registerTestParams) error {
			t.Fatal("handler must not be called on malformed payload")
			return nil
		})

	err := env.handler(context.Background(), []byte(`not-json`))

	require.Error(t, err)
	require.ErrorContains(t, err, "test_job")
}

func TestRegisterEnqueueFuncBuildsBackgroundTask(t *testing.T) {
	env := &registerTestEnv{}

	enqueue := Register(env, model.BackgroundTelegramJobQueue, "test_job", 7,
		func(_ context.Context, _ registerTestParams) error { return nil })

	params := registerTestParams{Value: "payload"}
	err := enqueue(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, 1, env.enqueueCallCount)
	require.Equal(t, model.BackgroundTask{
		ID:       "test_job",
		Queue:    model.BackgroundTelegramJobQueue,
		Data:     params,
		Priority: 7,
	}, env.enqueuedTask)
}
