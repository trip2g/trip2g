package agentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// goldenHashOfHello pins fleet's contentHash to trip2g's hashContent
// (internal/case/updatenotes). The same constant is asserted from the trip2g
// side in that package's tests: the two implementations are independent, and a
// silent drift would turn every conditional patch into a hash mismatch — a
// failure that would look like a concurrency problem, not like a bug here.
const goldenHashOfHello = "LPJNul-wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ="

func TestContentHashGolden(t *testing.T) {
	require.Equal(t, goldenHashOfHello, contentHash("hello"))
}

type conditionalKB struct {
	KB
	gotHash   string
	gotCalled bool
	plainCall bool
}

func (c *conditionalKB) PatchIfUnchanged(_ context.Context, _, _, _, expectedHash string) error {
	c.gotCalled = true
	c.gotHash = expectedHash
	return nil
}

func (c *conditionalKB) Patch(context.Context, string, string, string) error {
	c.plainCall = true
	return nil
}

// The hash must be of the exact bytes the guard verified, so trip2g rejects the
// patch if anything moved in between.
func TestScopedKB_PatchIsConditionalOnVerifiedBytes(t *testing.T) {
	const body = "hello world\n"
	kb := &conditionalKB{KB: newMemKB(map[string]string{"notes/plain.md": body})}
	scoped := NewScopedKB(kb, nil, []string{"**"})

	require.NoError(t, scoped.Patch(context.Background(), "notes/plain.md", "world", "there"))

	require.True(t, kb.gotCalled, "a hash-capable KB must get the conditional patch")
	require.False(t, kb.plainCall, "the unconditional path must not be used")
	require.Equal(t, contentHash(body), kb.gotHash)
}

// With the guard off there is no verification read, so there is no hash to
// condition on and the plain path is correct.
func TestScopedKB_PatchUnconditionalWhenGuardOff(t *testing.T) {
	kb := &conditionalKB{KB: newMemKB(map[string]string{"notes/plain.md": "hello\n"})}
	scoped := NewScopedKB(kb, nil, []string{"**"})
	scoped.allowRoleAuthoring = true

	require.NoError(t, scoped.Patch(context.Background(), "notes/plain.md", "hello", "bye"))

	require.False(t, kb.gotCalled)
	require.True(t, kb.plainCall)
}

// A KB that cannot do conditional patches still works.
func TestScopedKB_PatchFallsBackWithoutConditionalSupport(t *testing.T) {
	kb := newMemKB(map[string]string{"notes/plain.md": "hello world\n"})
	scoped := NewScopedKB(kb, nil, []string{"**"})

	require.NoError(t, scoped.Patch(context.Background(), "notes/plain.md", "world", "there"))

	got, err := kb.Read(context.Background(), "notes/plain.md")
	require.NoError(t, err)
	require.Equal(t, "hello there\n", got)
}
