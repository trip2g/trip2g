package coderun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"trip2g/internal/webhookutil"
)

// CodeInput is the parameters for executing the fenced code in a rendered body.
// Body must be pre-rendered (Jet-evaluated by the caller); the executor extracts
// the fenced code blocks from the rendered text. Scope enforcement (KB writes,
// write_patterns) is NOT part of code execution — it stays with the caller
// (fleet's agentruntime.RunCode applies the returned changes through ScopedKB).
type CodeInput struct {
	Body            string        // Jet-rendered role body
	Program         string        // optional override; empty → derived from fence language
	AllowedPrograms []string      // fleet allowlist; empty → code execution disabled
	Timeout         time.Duration // per-run timeout; 0 → bounded by ctx only
	Input           []byte        // delivery bag JSON written to $FLEET_INPUT
	EnvPassthrough  []string      // exact parent env var names forwarded to child
	EnvPrefix       []string      // parent env var name prefixes forwarded to child
	MaxStdoutBytes  int           // stdout cap per code child; 0 → 1 MiB default
	Sandbox         SandboxPolicy // OS-level isolation; zero value = safe default (native)
}

// ExecCode extracts and runs the fenced blocks in in.Body and returns the parsed
// {changes, answer} stdout contract WITHOUT applying the changes to a KB and
// WITHOUT scope enforcement. codellm uses it to map the changes onto OpenAI
// tool_calls, leaving scope enforcement (write_patterns via ScopedKB) to the
// caller (fleet). A block execution failure or a stdout-parse failure returns an
// error, which codellm maps to HTTP 422 (deterministic hard-fail, no retry).
func ExecCode(ctx context.Context, in CodeInput) ([]webhookutil.AgentChange, string, error) {
	core, err := Exec(ctx, in, false)
	return core.Changes, core.Answer, err
}

// BlockDebug is one block's captured execution, exposed to the debugger surface
// (the x_fleet_debug extension on codellm's /v1/chat/completions). Stdout is the
// block's own stdout; PipeBuffer is the inter-block buffer this block feeds into
// the NEXT block's stdin (empty for the last block, which has no downstream).
type BlockDebug struct {
	Index      int
	Stdout     string
	PipeBuffer string
}

// ExecCodeDebug is ExecCode plus per-block capture for the debugger. It drives
// the pipelineDebugTaps seam with a non-nil tap per inter-block edge (via the
// io.TeeReader path in wirePipeline), so each block's stdout / inter-block pipe
// buffer is captured as it streams. Otherwise identical to ExecCode (same
// {changes, answer} contract, same hard-fail semantics).
func ExecCodeDebug(ctx context.Context, in CodeInput) ([]webhookutil.AgentChange, string, []BlockDebug, error) {
	core, err := Exec(ctx, in, true)
	return core.Changes, core.Answer, core.Debug, err
}

// ExecResult is the execution core's result: the parsed {changes, answer}
// contract, the block count (Steps), and — when capture was requested — the
// per-block debug capture.
type ExecResult struct {
	Changes []webhookutil.AgentChange
	Answer  string
	Steps   int
	Debug   []BlockDebug // nil unless capture was requested
}

// Exec is the shared execution core of ExecCode, ExecCodeDebug, and fleet's
// agentruntime.RunCode: it extracts the fenced blocks, resolves+allowlist-checks
// their programs, runs them (single block via RunBlock, multiple via the
// streaming pipeline), and parses the last block's stdout as the {changes,
// answer} contract. It performs no KB writes and no scope checks. When capture is
// true it also returns per-block BlockDebug via a non-nil pipelineDebugTaps; when
// false no taps are allocated (the zero-overhead production path).
func Exec(ctx context.Context, in CodeInput, capture bool) (ExecResult, error) {
	blocks := ExtractFencedBlocks(in.Body)
	if len(blocks) == 0 {
		return ExecResult{}, errors.New("coderun: no fenced code block found in rendered body")
	}

	programs, err := resolvePrograms(blocks, in)
	if err != nil {
		return ExecResult{}, err
	}

	limit := in.MaxStdoutBytes
	if limit == 0 {
		limit = 1 << 20
	}

	var stdout string
	var debug []BlockDebug
	if len(blocks) == 1 {
		stdout, debug, err = runSingleBlock(ctx, blocks[0], programs[0], in, limit, capture)
	} else {
		stdout, debug, err = runMultiBlock(ctx, blocks, programs, in, limit, capture)
	}
	if err != nil {
		return ExecResult{}, err
	}

	rawChanges, answer, perr := ParseCodeOutput(stdout)
	if perr != nil {
		return ExecResult{}, perr
	}
	return ExecResult{Changes: rawChanges, Answer: answer, Steps: len(blocks), Debug: debug}, nil
}

// runSingleBlock runs exactly one fenced block via RunBlock for byte-identical
// behavior with the legacy single-block path. debug is non-nil only when capture.
func runSingleBlock(
	ctx context.Context, block FencedBlock, program string, in CodeInput, limit int, capture bool,
) (string, []BlockDebug, error) {
	out, _, _, runErr := RunBlock(ctx, CodeSpec{
		Program:        program,
		Code:           block.Code,
		Input:          in.Input,
		Timeout:        in.Timeout,
		EnvPassthrough: in.EnvPassthrough,
		EnvPrefix:      in.EnvPrefix,
		MaxStdoutBytes: limit,
		Sandbox:        in.Sandbox,
	})
	if runErr != nil {
		return "", nil, fmt.Errorf("coderun: %w", runErr)
	}
	var debug []BlockDebug
	if capture {
		debug = []BlockDebug{{Index: 0, Stdout: out}}
	}
	return out, debug, nil
}

// runMultiBlock runs 2+ blocks through the streaming pipeline. In debug mode it
// allocates a tap per inter-block edge so wirePipeline tees each block's stdout
// into it. debug is non-nil only when capture.
func runMultiBlock(
	ctx context.Context, blocks []FencedBlock, programs []string, in CodeInput, limit int, capture bool,
) (string, []BlockDebug, error) {
	var taps pipelineDebugTaps
	if capture {
		taps = make(pipelineDebugTaps, len(blocks)-1)
		for i := range taps {
			taps[i] = &limitedBuffer{limit: limit}
		}
	}
	out, pipeErr := runPipeline(ctx, blocks, programs, in, limit, taps)
	if pipeErr != nil {
		return "", nil, pipeErr
	}
	var debug []BlockDebug
	if capture {
		debug = buildBlockDebug(taps, out, len(blocks))
	}
	return out, debug, nil
}

// buildBlockDebug assembles per-block debug from the captured inter-block taps
// (n-1 of them) plus the last block's final stdout. Block i<n-1 streams its
// stdout into the pipe feeding block i+1, so its Stdout and PipeBuffer are the
// same captured tap; the last block's Stdout is the final output and its
// PipeBuffer is empty (no downstream).
func buildBlockDebug(taps pipelineDebugTaps, finalStdout string, n int) []BlockDebug {
	debug := make([]BlockDebug, n)
	for i := range n {
		d := BlockDebug{Index: i}
		if i < n-1 {
			buf := taps[i].String()
			d.Stdout = buf
			d.PipeBuffer = buf
		} else {
			d.Stdout = finalStdout
		}
		debug[i] = d
	}
	return debug
}

// resolvePrograms resolves and allowlist-checks every block's program before
// any block runs: a disallowed later block must fail the whole run up front.
func resolvePrograms(blocks []FencedBlock, in CodeInput) ([]string, error) {
	programs := make([]string, len(blocks))
	for i, b := range blocks {
		program := in.Program
		if program == "" {
			program = fenceLangToProgram(b.Lang)
		}
		if program == "" {
			return nil, fmt.Errorf("coderun: %sfence language %q not supported (use python, bash, or node)", blockPrefix(i, len(blocks)), b.Lang)
		}
		if !IsAllowed(program, in.AllowedPrograms) {
			if len(in.AllowedPrograms) == 0 {
				return nil, errors.New("coderun: code execution disabled (set --allowed-programs)")
			}
			return nil, fmt.Errorf("coderun: %sprogram %q not in --allowed-programs %v", blockPrefix(i, len(blocks)), program, in.AllowedPrograms)
		}
		programs[i] = program
	}
	return programs, nil
}

// pipelineDebugTaps optionally captures each inter-block stdout for debug
// stepping. nil in the production path (ExecCode) — no allocation overhead.
// The debug endpoint passes a pre-allocated slice of capped buffers when it
// wants to capture intermediate outputs via io.TeeReader.
type pipelineDebugTaps []*limitedBuffer

// builtBlock is a fully prepared (not yet started) pipeline stage.
type builtBlock struct {
	cmd       *exec.Cmd
	workDir   string
	blockCtx  context.Context
	cancel    context.CancelFunc
	errBuf    limitedBuffer
	sandboxed bool
}

// runPipeline runs len(blocks) > 1 as a true streaming pipeline: block i's
// stdout feeds block i+1's stdin via io.Pipe, all blocks run concurrently.
// Sandbox decisions are resolved for ALL blocks before any starts (enforcing
// mode refuses the whole run if any block's sandbox can't be built).
//
// debugTaps is nil in the production path. When non-nil (debug endpoint only),
// debugTaps[i] receives block i's stdout through io.TeeReader as it streams to
// block i+1's stdin; the last block always goes to a capped buffer regardless.
//
// Returns the last block's captured stdout, or the earliest-index block error.
func runPipeline(ctx context.Context, blocks []FencedBlock, programs []string, in CodeInput, limit int, debugTaps pipelineDebugTaps) (string, error) {
	n := len(blocks)

	built, err := buildPipelineBlocks(ctx, blocks, programs, in, limit)
	if err != nil {
		return "", err
	}
	defer cleanupPipeline(built)

	pipes := wirePipeline(built, limit, debugTaps)
	defer closeAllPipes(pipes)

	if startErr := startAll(built, pipes, n); startErr != nil {
		return "", startErr
	}

	return waitPipeline(built, pipes, in.Sandbox, n)
}

// buildPipelineBlocks prepares all blocks upfront — any build failure tears
// down already-built workdirs before returning, so no partial workdir leak.
func buildPipelineBlocks(ctx context.Context, blocks []FencedBlock, programs []string, in CodeInput, limit int) ([]builtBlock, error) {
	n := len(blocks)
	built := make([]builtBlock, 0, n)
	for i, b := range blocks {
		bb, err := buildBlockCmd(ctx, b.Code, programs[i], in)
		if err != nil {
			cleanupPipeline(built)
			return nil, fmt.Errorf("coderun: %s%w", blockPrefix(i, n), err)
		}
		bb.errBuf.limit = limit
		built = append(built, bb)
	}
	return built, nil
}

// wirePipeline connects consecutive blocks via io.Pipe and wires stdout/stderr;
// returns the slice of pipe writers (one per inter-block connection).
// debugTaps[i] — when non-nil — captures block i's stdout via TeeReader.
func wirePipeline(built []builtBlock, limit int, debugTaps pipelineDebugTaps) []*io.PipeWriter {
	n := len(built)
	pipes := make([]*io.PipeWriter, n-1)
	var lastBuf limitedBuffer
	lastBuf.limit = limit

	for i := range n - 1 {
		pr, pw := io.Pipe()
		pipes[i] = pw
		var reader io.Reader = pr
		if debugTaps != nil && debugTaps[i] != nil {
			reader = io.TeeReader(pr, debugTaps[i])
		}
		built[i].cmd.Stdout = pw
		built[i+1].cmd.Stdin = reader
	}
	built[n-1].cmd.Stdout = &lastBuf
	for i := range built {
		built[i].cmd.Stderr = &built[i].errBuf
	}
	// lastBuf is captured via the cmd.Stdout pointer; Go escapes its address
	// automatically when taken (cmd.Stdout = &lastBuf), so waitPipeline can read
	// it back after Wait completes.
	return pipes
}

// startAll starts all built blocks. On failure, closes already-opened upstream
// pipes with the start error so blocked readers drain.
func startAll(built []builtBlock, pipes []*io.PipeWriter, n int) error {
	for i := range built {
		if sErr := built[i].cmd.Start(); sErr != nil {
			for j := range i {
				_ = pipes[j].CloseWithError(sErr)
			}
			return fmt.Errorf("coderun: %sstart failed: %w", blockPrefix(i, n), sErr)
		}
	}
	return nil
}

// pipeWaitResult is one block's Wait() outcome.
type pipeWaitResult struct {
	idx int
	err error
}

// waitPipeline waits for all blocks, cancels on any failure, and returns the
// last block's stdout string or the earliest-index block error.
func waitPipeline(built []builtBlock, pipes []*io.PipeWriter, sandbox SandboxPolicy, n int) (string, error) {
	pipeCtx, pipeCancel := context.WithCancel(context.Background())
	defer pipeCancel()

	results := make(chan pipeWaitResult, n)
	for i := range built {
		go waitOne(i, built, pipes, n, sandbox, results)
	}

	errs := make([]error, n)
	stderrs := make([]string, n)
	for range n {
		r := <-results
		if r.err != nil {
			errs[r.idx] = r.err
			stderrs[r.idx] = built[r.idx].errBuf.String()
			pipeCancel()
		}
	}
	_ = pipeCtx // pipeCancel already deferred

	// Return the earliest-index block failure.
	for i, e := range errs {
		if e != nil {
			if built[i].blockCtx.Err() != nil {
				return "", fmt.Errorf("coderun: %stimed out", blockPrefix(i, n))
			}
			msg := stderrs[i]
			if msg == "" {
				msg = e.Error()
			}
			return "", fmt.Errorf("coderun: %snon-zero exit: %s", blockPrefix(i, n), msg)
		}
	}

	// All blocks succeeded; extract the last block's stdout via the Stdout pointer.
	lastBuf, ok := built[n-1].cmd.Stdout.(*limitedBuffer)
	if !ok || lastBuf == nil {
		return "", errors.New("coderun: internal error: last block stdout not captured")
	}
	return lastBuf.String(), nil
}

// waitOne waits for block idx and sends its result on ch. Closes the
// downstream pipe writer (if any) so the next block sees EOF.
func waitOne(idx int, built []builtBlock, pipes []*io.PipeWriter, n int, sandbox SandboxPolicy, ch chan<- pipeWaitResult) {
	werr := built[idx].cmd.Wait()
	if idx < n-1 {
		if werr != nil {
			_ = pipes[idx].CloseWithError(werr)
		} else {
			_ = pipes[idx].Close()
		}
	}
	// BestEffort mode: sandboxed child failed to start (ProcessState == nil).
	// We can't fall back mid-pipeline — report it clearly.
	if werr != nil && built[idx].sandboxed && built[idx].cmd.ProcessState == nil && built[idx].blockCtx.Err() == nil {
		if !sandbox.Mode.enforcing() {
			werr = fmt.Errorf("sandboxed child failed to start; pipeline cannot fall back mid-run: %w", werr)
		}
	}
	ch <- pipeWaitResult{idx, werr}
}

// cleanupPipeline cancels all block contexts and removes all workdirs.
func cleanupPipeline(built []builtBlock) {
	for _, bb := range built {
		bb.cancel()
		_ = os.RemoveAll(bb.workDir)
	}
}

// closeAllPipes closes all inter-block pipe writers. Safe to call after Wait.
func closeAllPipes(pipes []*io.PipeWriter) {
	for _, pw := range pipes {
		if pw != nil {
			_ = pw.Close()
		}
	}
}

// buildBlockCmd prepares one pipeline block: creates a throwaway workdir,
// writes the code + input files, builds the scrubbed env, and resolves the
// sandbox decision (enforcing: fail here on build error; bestEffort: fall back
// to unsandboxed now so no rebuild is needed mid-stream).
// The returned cmd has no Stdin/Stdout/Stderr wired — the caller does that.
// The caller must call cancel() and os.RemoveAll(workDir) after the cmd exits.
func buildBlockCmd(ctx context.Context, code, program string, in CodeInput) (builtBlock, error) {
	interp, ok := currentRegistry().byName[program]
	if !ok {
		return builtBlock{}, fmt.Errorf("unknown program %q", program)
	}

	workDir, mkErr := os.MkdirTemp("", "fleet-coderun-*")
	if mkErr != nil {
		return builtBlock{}, fmt.Errorf("create workdir: %w", mkErr)
	}
	tear := func() { _ = os.RemoveAll(workDir) }

	codeFile := filepath.Join(workDir, "run"+interp.Ext)
	if wErr := os.WriteFile(codeFile, []byte(code), 0o600); wErr != nil {
		tear()
		return builtBlock{}, fmt.Errorf("write code file: %w", wErr)
	}

	inputData := in.Input
	if len(inputData) == 0 {
		inputData = []byte("{}")
	}
	inputFile := filepath.Join(workDir, "input.json")
	if wErr := os.WriteFile(inputFile, inputData, 0o600); wErr != nil {
		tear()
		return builtBlock{}, fmt.Errorf("write input file: %w", wErr)
	}

	env := buildChildEnv(inputFile, in.EnvPassthrough, in.EnvPrefix)
	fullArgv := append(append([]string{}, interp.Cmd...), codeFile)

	blockCtx := ctx
	cancel := context.CancelFunc(func() {})
	if in.Timeout > 0 {
		blockCtx, cancel = context.WithTimeout(ctx, in.Timeout)
	}

	policy := in.Sandbox.withDefaults(in.Timeout)
	sandboxed := policy.Mode != SandboxOff

	if sandboxed && !sandboxSupportedOS {
		if policy.Mode.enforcing() {
			cancel()
			tear()
			return builtBlock{}, fmt.Errorf("sandbox mode %q requires Linux; refusing to run unsandboxed", policy.Mode)
		}
		warnSandboxFallback("unsupported OS", nil)
		sandboxed = false
	}

	var cmd *exec.Cmd
	if sandboxed {
		sbCmd, sbErr := sandboxCommand(blockCtx, fullArgv, workDir, policy)
		if sbErr != nil {
			if policy.Mode.enforcing() {
				cancel()
				tear()
				return builtBlock{}, fmt.Errorf("sandbox setup failed: %w", sbErr)
			}
			warnSandboxFallback("sandbox setup failed", sbErr)
			sandboxed = false
		} else {
			cmd = sbCmd
		}
	}
	if !sandboxed {
		// gosec G204: operator-allowlisted interpreter on role's rendered code —
		// executing that code IS the feature, off by default. Same rationale as RunBlock.
		cmd = exec.CommandContext(blockCtx, fullArgv[0], fullArgv[1:]...) //nolint:gosec // allowlisted interpreter
	}
	cmd.Dir = workDir
	cmd.Env = env
	// Caller wires Stdin/Stdout/Stderr and calls cmd.Start().

	return builtBlock{
		cmd:       cmd,
		workDir:   workDir,
		blockCtx:  blockCtx,
		cancel:    cancel,
		sandboxed: sandboxed,
	}, nil
}

// blockPrefix returns "block i/n: " for multi-block pipelines and "" for a
// single block, keeping single-block error messages unchanged.
func blockPrefix(i, n int) string {
	if n <= 1 {
		return ""
	}
	return fmt.Sprintf("block %d/%d: ", i+1, n)
}

// IsAllowed reports whether program is in the allowedPrograms list.
// Both program and list entries use canonical names (python, bash, node).
func IsAllowed(program string, allowedPrograms []string) bool {
	for _, p := range allowedPrograms {
		if p == program {
			return true
		}
	}
	return false
}
