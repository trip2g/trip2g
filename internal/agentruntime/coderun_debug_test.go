package agentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExecCodeDebug_CapturesInterBlockBuffer drives the pipelineDebugTaps seam
// with a real two-block pipeline and asserts the tap captured the inter-block
// pipe buffer (block 0's stdout that feeds block 1's stdin). This is the test
// the seam previously lacked — before this, no caller ever passed a non-nil tap,
// so the io.TeeReader path in wirePipeline was dead code.
func TestExecCodeDebug_CapturesInterBlockBuffer(t *testing.T) {
	skipIfSandboxUnsupported(t)

	// Block 0 emits "hello"; block 1 consumes ALL of stdin and echoes it back as
	// the answer, so the whole inter-block buffer flows through the tap.
	body := "```bash\necho hello\n```\n" +
		"```bash\nv=$(cat); echo \"{\\\"changes\\\":[],\\\"answer\\\":\\\"$v\\\"}\"\n```"

	changes, answer, debug, err := ExecCodeDebug(context.Background(), CodeInput{
		Body:            body,
		AllowedPrograms: []string{"bash"},
	})
	require.NoError(t, err)
	require.Empty(t, changes)
	require.Equal(t, "hello", answer)

	require.Len(t, debug, 2, "one BlockDebug per block")

	// Block 0: its stdout is the inter-block pipe buffer feeding block 1.
	require.Equal(t, 0, debug[0].Index)
	require.Equal(t, "hello\n", debug[0].PipeBuffer, "tap must capture block 0's stdout")
	require.Equal(t, "hello\n", debug[0].Stdout)

	// Block 1 (last): stdout is the final output, no downstream pipe buffer.
	require.Equal(t, 1, debug[1].Index)
	require.Empty(t, debug[1].PipeBuffer, "last block has no downstream pipe")
	require.Contains(t, debug[1].Stdout, `"answer":"hello"`)
}

// TestExecCodeDebug_SingleBlock asserts the single-block debug path: one block,
// its stdout captured, no inter-block pipe buffer.
func TestExecCodeDebug_SingleBlock(t *testing.T) {
	skipIfSandboxUnsupported(t)

	body := "```bash\necho '{\"changes\":[],\"answer\":\"solo\"}'\n```"
	_, answer, debug, err := ExecCodeDebug(context.Background(), CodeInput{
		Body:            body,
		AllowedPrograms: []string{"bash"},
	})
	require.NoError(t, err)
	require.Equal(t, "solo", answer)
	require.Len(t, debug, 1)
	require.Equal(t, 0, debug[0].Index)
	require.Empty(t, debug[0].PipeBuffer)
	require.Contains(t, debug[0].Stdout, `"answer":"solo"`)
}
