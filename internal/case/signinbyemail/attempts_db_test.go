package signinbyemail_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"trip2g/internal/case/signinbyemail"
	"trip2g/internal/db"
	gmodel "trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

// dbEnv runs the use case on a real SQLite write connection without a request
// transaction, so concurrent guesses interleave statement by statement.
type dbEnv struct {
	*db.WriteQueries

	// checked counts guesses that were compared against a code still under the limit.
	checked atomic.Int64
}

func (e *dbEnv) EmailSignInEnabled(context.Context) bool { return true }
func (e *dbEnv) DevSignInBypass(string) bool             { return false }

func (e *dbEnv) SetupUserToken(context.Context, int64) (string, error) {
	return "tok", nil
}

func (e *dbEnv) SpendSignInCodeAttempt(ctx context.Context, email *string) ([]db.SpendSignInCodeAttemptRow, error) {
	rows, err := e.WriteQueries.SpendSignInCodeAttempt(ctx, email)
	for _, r := range rows {
		if r.FailedAttempts <= 5 {
			e.checked.Add(1)
			break
		}
	}
	return rows, err
}

func TestParallelWrongGuessesCannotExceedLimit(t *testing.T) {
	conn, err := db.Setup(db.SetupConfig{
		DatabaseFile: filepath.Join(t.TempDir(), "signin.sqlite3"),
		SkipDump:     true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	env := &dbEnv{WriteQueries: db.NewWriteQueries(conn)}

	user, err := env.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{
		Email:      "victim@example.com",
		CreatedVia: "test",
	})
	require.NoError(t, err)

	err = env.InsertSignInCode(ctx, db.InsertSignInCodeParams{UserID: user.ID, Code: "123456"})
	require.NoError(t, err)

	const guesses = 30

	var wg sync.WaitGroup
	for i := range guesses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, resolveErr := signinbyemail.Resolve(ctx, env, gmodel.SignInByEmailInput{
				Email: "victim@example.com",
				Code:  fmt.Sprintf("%06d", 900000+i),
			})
			if resolveErr != nil {
				t.Error(resolveErr)
				return
			}
			if _, ok := res.(*gmodel.ErrorPayload); !ok {
				t.Errorf("wrong guess got %T", res)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int64(5), env.checked.Load())

	res, err := signinbyemail.Resolve(ctx, env, gmodel.SignInByEmailInput{
		Email: "victim@example.com",
		Code:  "123456",
	})
	require.NoError(t, err)
	require.IsType(t, &gmodel.ErrorPayload{}, res, "the correct code must be dead after the limit")

	active, err := env.CountActiveSignInCodes(ctx, user.ID)
	require.NoError(t, err)
	require.Zero(t, active)
}
