package jsonneteval_test

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/jsonneteval"
)

func runawayPrograms() map[string]string {
	return map[string]string{
		"tailstrict":   `local loop(i, acc) = if i == 0 then acc else loop(i - 1, acc + 1) tailstrict; loop(200000, 0)`,
		"super chain":  `local o = std.foldl(function(acc, x) acc + {a: super.a + 1}, std.range(1, 3000000), {a: 0}); o.a`,
		"object chain": `local o = std.foldl(function(acc, x) acc {b: 1}, std.range(1, 3000000), {a: 0}); o.a`,
	}
}

// A Go stack overflow is fatal and would take the test binary down with it,
// so each program runs in a child process.
func TestEvalJSON_RunawayIsAnError(t *testing.T) {
	for name := range runawayPrograms() {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestEvalJSON_RunawayChild$", "-test.v")
			cmd.Env = append(os.Environ(), "JSONNETEVAL_RUNAWAY="+name)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "child died:\n%.2000s", out)
			require.Contains(t, string(out), "--- PASS: TestEvalJSON_RunawayChild")
		})
	}
}

func TestEvalJSON_RunawayChild(t *testing.T) {
	name := os.Getenv("JSONNETEVAL_RUNAWAY")
	if name == "" {
		t.Skip("run by TestEvalJSON_RunawayIsAnError")
	}

	_, err := jsonneteval.EvalJSON(runawayPrograms()[name], nil)
	require.Error(t, err)
}

func TestVM_Timeout(t *testing.T) {
	vm := jsonneteval.NewVM()
	vm.Timeout = 50 * time.Millisecond

	start := time.Now()
	_, err := vm.EvaluateAnonymousSnippet("slow", `std.length([x for x in std.range(1, 3000) for y in std.range(1, 3000)])`)
	require.ErrorIs(t, err, jsonneteval.ErrTimeout)
	require.Less(t, time.Since(start), 2*time.Second)
}

func TestVM_TooDeep(t *testing.T) {
	vm := jsonneteval.NewVM()

	_, err := vm.EvaluateAnonymousSnippet("deep", runawayPrograms()["tailstrict"])
	require.ErrorIs(t, err, jsonneteval.ErrTooDeep)
}

func TestVM_TooManyAdditions(t *testing.T) {
	vm := jsonneteval.NewVM()

	_, err := vm.EvaluateAnonymousSnippet("chain", runawayPrograms()["object chain"])
	require.ErrorIs(t, err, jsonneteval.ErrTooManyAdditions)
}

func TestVM_BudgetIsPerEvaluation(t *testing.T) {
	vm := jsonneteval.NewVM()

	_, err := vm.EvaluateAnonymousSnippet("deep", runawayPrograms()["tailstrict"])
	require.ErrorIs(t, err, jsonneteval.ErrTooDeep)

	out, err := vm.EvaluateAnonymousSnippet("loop", `local loop(i, acc) = if i == 0 then acc else loop(i - 1, acc + 1) tailstrict; loop(1000, 0)`)
	require.NoError(t, err)
	require.Equal(t, "1000\n", out)
}
