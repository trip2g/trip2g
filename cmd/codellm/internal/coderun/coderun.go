package coderun

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"trip2g/internal/webhookutil"
)

//go:embed interpreters.json
var defaultInterpretersJSON []byte

// interpreter describes one language runtime: binary argv prefix, fence labels,
// temp-file extension, and the env it needs to find its packages.
type interpreter struct {
	Name            string   `json:"name"`
	Cmd             []string `json:"cmd"`
	CodeBlockLabels []string `json:"code_block_labels"`
	Ext             string   `json:"ext"`
	Env             []envVar `json:"env"`
}

// envVar is one declarative child-env entry. IfExists names a path that must be
// present for the var to be set, so an entry describing a path baked into the
// runtime image is simply skipped in a dev run outside it — pointing an
// interpreter at a missing bundle breaks it rather than configures it.
type envVar struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	IfExists string `json:"if_exists"`
}

// interpreterRegistry holds two indices built from the loaded interpreter list,
// plus the env applied to every interpreter.
type interpreterRegistry struct {
	byLabel map[string]*interpreter
	byName  map[string]*interpreter
	env     []envVar
}

// registry is guarded by registryMu: a startup --interpreters override may race
// with concurrent role runs. Each *interpreterRegistry is immutable after build;
// only the pointer swap needs the lock.
//
//nolint:gochecknoglobals // interpreter lookup built from embedded config (overridable via --interpreters); package-level by design
var (
	registryMu sync.RWMutex
	registry   = mustBuildRegistry(defaultInterpretersJSON)
)

// currentRegistry returns the active immutable registry snapshot.
func currentRegistry() *interpreterRegistry {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry
}

// mustBuildRegistry panics if data cannot be parsed. Used at package init.
func mustBuildRegistry(data []byte) *interpreterRegistry {
	r, err := buildRegistry(data)
	if err != nil {
		panic("coderun: failed to build interpreter registry: " + err.Error())
	}
	return r
}

// buildRegistry parses an interpreters JSON blob and builds byLabel and byName indices.
func buildRegistry(data []byte) (*interpreterRegistry, error) {
	var payload struct {
		Env          []envVar      `json:"env"`
		Interpreters []interpreter `json:"interpreters"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse interpreters JSON: %w", err)
	}
	reg := &interpreterRegistry{
		byLabel: make(map[string]*interpreter),
		byName:  make(map[string]*interpreter),
		env:     payload.Env,
	}
	for i := range payload.Interpreters {
		interp := &payload.Interpreters[i]
		reg.byName[interp.Name] = interp
		for _, label := range interp.CodeBlockLabels {
			reg.byLabel[label] = interp
		}
	}
	return reg, nil
}

// SetInterpretersJSON replaces the runtime registry with one built from data.
func SetInterpretersJSON(data []byte) error {
	r, err := buildRegistry(data)
	if err != nil {
		return err
	}
	registryMu.Lock()
	registry = r
	registryMu.Unlock()
	return nil
}

// LoadInterpretersFile reads a JSON file and rebuilds the interpreter registry.
func LoadInterpretersFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("LoadInterpretersFile: %w", err)
	}
	return SetInterpretersJSON(data)
}

// FenceLangKnown reports whether a Markdown fence label is registered.
func FenceLangKnown(lang string) bool {
	_, ok := currentRegistry().byLabel[strings.ToLower(lang)]
	return ok
}

// CodeSpec is the parameters for one code-execution call.
type CodeSpec struct {
	Program        string         // canonical program name (python, bash, node, ...)
	Code           string         // source text written to a per-run temp file
	Stdin          []byte         // fed to the child's stdin (pipeline: prior block's stdout); nil → /dev/null
	Input          []byte         // delivery bag JSON; nil or empty → "{}"
	Timeout        time.Duration  // per-run timeout; 0 = bounded by ctx only
	EnvPassthrough []string       // exact parent env var names to forward to child
	EnvPrefix      []string       // parent env var name prefixes to forward to child
	MaxStdoutBytes int            // stdout cap; 0 → 1 MiB default
	Sandbox        SandboxPolicy  // OS-level isolation; zero value = safe default (native)
	Stats          *RunBlockStats // optional execution metrics sink
}

type RunBlockStats struct {
	DurationMs  int64
	MaxRSSBytes int64
}

// codeOutput is the stdout JSON contract that every code program must emit.
type codeOutput struct {
	Changes []rawCodeChange `json:"changes"`
	Answer  string          `json:"answer"`
}

// rawCodeChange is one entry in the stdout changes array.
type rawCodeChange struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Find    string `json:"find,omitempty"`
	Replace string `json:"replace,omitempty"`
}

// RunBlock executes spec.Code under spec.Program in a throwaway temp workdir.
//
// Secret-scrub is a HARD security invariant: cmd.Env is set to a minimal
// allowlist (PATH + FLEET_INPUT only). The parent process env is NEVER
// inherited. Passing nil to cmd.Env inherits the parent env (os/exec default),
// which exposes JWT secrets and LLM API keys to the child; we never do that.
//
// FLEET_INPUT points to a temp file containing spec.Input (the delivery bag
// JSON) so the child can read structured trigger data. The scoped write token
// is not in the bag or the env.
//
// stdout is capped at MaxStdoutBytes (default 1 MiB); stderr is also captured for diagnostics.
// Non-zero exit or context timeout returns an error; timedOut distinguishes
// the two failure modes.
func RunBlock(ctx context.Context, spec CodeSpec) (string, string, bool, error) {
	interp, ok := currentRegistry().byName[spec.Program]
	if !ok {
		return "", "", false, fmt.Errorf("coderun: unknown program %q", spec.Program)
	}

	// Per-run throwaway workdir: prevents cross-run contamination.
	workDir, mkErr := os.MkdirTemp("", "fleet-coderun-*")
	if mkErr != nil {
		return "", "", false, fmt.Errorf("coderun: create workdir: %w", mkErr)
	}
	defer os.RemoveAll(workDir)

	codeFile := filepath.Join(workDir, "run"+interp.Ext)
	if wErr := os.WriteFile(codeFile, []byte(spec.Code), 0o600); wErr != nil {
		return "", "", false, fmt.Errorf("coderun: write code file: %w", wErr)
	}

	inputData := spec.Input
	if len(inputData) == 0 {
		inputData = []byte("{}")
	}
	inputFile := filepath.Join(workDir, "input.json")
	if wErr := os.WriteFile(inputFile, inputData, 0o600); wErr != nil {
		return "", "", false, fmt.Errorf("coderun: write input file: %w", wErr)
	}

	runCtx := ctx
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}

	// gosec G204 (excluded for this file in .golangci): runs an operator-
	// allowlisted interpreter (allowed_programs) on the role's rendered code —
	// executing that code IS the feature, and it's off by default.
	fullArgv := append(append([]string{}, interp.Cmd...), codeFile)
	// Secret-scrub: build an explicit MINIMAL env. NEVER nil (inherits parent)
	// or os.Environ() (exposes all secrets). The base is PATH + FLEET_INPUT only.
	// Selective pass-through adds only explicitly declared vars/prefixes on top.
	env := buildChildEnv(inputFile, interp, spec.EnvPassthrough, spec.EnvPrefix)
	limit := spec.MaxStdoutBytes
	if limit == 0 {
		limit = 1 << 20
	}

	policy := spec.Sandbox.withDefaults(spec.Timeout)
	sandboxed := policy.Mode != SandboxOff

	// Fail-closed: when the enforcing mode is requested but the OS cannot
	// sandbox at all, REFUSE the run — never run untrusted code unconfined.
	if sandboxed && !sandboxSupportedOS {
		if policy.Mode.enforcing() {
			return "", "", false, fmt.Errorf("coderun: sandbox mode %q requires Linux; refusing to run unsandboxed", policy.Mode)
		}
		warnSandboxFallback("unsupported OS", nil)
		sandboxed = false
	}

	var cmd *exec.Cmd
	if sandboxed {
		sbCmd, sbErr := sandboxCommand(runCtx, fullArgv, workDir, policy)
		if sbErr != nil {
			// Enforcing mode: a sandbox that cannot be built refuses the run.
			if policy.Mode.enforcing() {
				return "", "", false, fmt.Errorf("coderun: sandbox setup failed: %w", sbErr)
			}
			warnSandboxFallback("sandbox setup failed", sbErr)
			sandboxed = false
		} else {
			cmd = sbCmd
		}
	}
	if !sandboxed {
		cmd = exec.CommandContext(runCtx, fullArgv[0], fullArgv[1:]...)
	}
	var outBuf, errBuf limitedBuffer
	prepareCmd(cmd, workDir, env, spec.Stdin, &outBuf, &errBuf, limit)

	startedAt := time.Now()
	runErr := cmd.Run()
	if spec.Stats != nil {
		spec.Stats.DurationMs = time.Since(startedAt).Milliseconds()
		spec.Stats.MaxRSSBytes = maxRSSBytes(cmd.ProcessState)
	}
	// A sandboxed command that failed to START never ran the child: namespace
	// creation was denied (e.g. unprivileged userns disabled) or a confinement
	// step failed. Enforcing mode REFUSES the run (the code never ran, so this
	// is a clean failure, not a partial one). BestEffort degrades to plain
	// execution with a per-run warning.
	if runErr != nil && sandboxed && cmd.ProcessState == nil && runCtx.Err() == nil {
		if policy.Mode.enforcing() {
			return "", "", false, fmt.Errorf("coderun: sandboxed child failed to start; refusing to run unsandboxed: %w", runErr)
		}
		warnSandboxFallback("sandboxed child failed to start", runErr)
		cmd = exec.CommandContext(runCtx, fullArgv[0], fullArgv[1:]...)
		prepareCmd(cmd, workDir, env, spec.Stdin, &outBuf, &errBuf, limit)
		runErr = cmd.Run()
	}
	outStr := outBuf.String()
	errStr := errBuf.String()

	if runCtx.Err() != nil {
		return outStr, errStr, true, errors.New("coderun: timed out")
	}
	if runErr != nil {
		msg := errStr
		if msg == "" {
			msg = runErr.Error()
		}
		return outStr, errStr, false, fmt.Errorf("coderun: non-zero exit: %s", msg)
	}
	return outStr, errStr, false, nil
}

// prepareCmd applies the run wiring shared by the sandboxed and plain paths:
// workdir, scrubbed env, stdin, and fresh capped stdout/stderr buffers.
func prepareCmd(cmd *exec.Cmd, workDir string, env []string, stdin []byte, outBuf, errBuf *limitedBuffer, limit int) {
	cmd.Dir = workDir
	cmd.Env = env
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	*outBuf = limitedBuffer{limit: limit}
	*errBuf = limitedBuffer{limit: limit}
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
}

// ParseCodeOutput parses the code child's stdout JSON into []AgentChange +
// answer string. stdout must match {"changes":[...],"answer":"..."}.
// A "content"-only entry → AgentChangeKindWrite; "find"/"replace" → AgentChangeKindPatch.
func ParseCodeOutput(stdout string) ([]webhookutil.AgentChange, string, error) {
	var out codeOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		preview := stdout
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, "", fmt.Errorf("coderun: parse stdout: %w (got: %q)", err, preview)
	}
	changes := make([]webhookutil.AgentChange, 0, len(out.Changes))
	for i, c := range out.Changes {
		if c.Path == "" {
			return nil, "", fmt.Errorf("coderun: change[%d]: missing path", i)
		}
		if c.Find != "" || c.Replace != "" {
			changes = append(changes, webhookutil.AgentChange{
				Path:    c.Path,
				Find:    c.Find,
				Replace: c.Replace,
				Kind:    webhookutil.AgentChangeKindPatch,
			})
		} else {
			changes = append(changes, webhookutil.AgentChange{
				Path:    c.Path,
				Content: c.Content,
				Kind:    webhookutil.AgentChangeKindWrite,
			})
		}
	}
	return changes, out.Answer, nil
}

// programBinary resolves a program name to the binary to run via the interpreter
// registry. Returns an error for unrecognized names so the caller can fail fast.
func programBinary(name string) (string, error) {
	if interp, ok := currentRegistry().byName[name]; ok {
		return interp.Cmd[0], nil
	}
	return "", fmt.Errorf("coderun: unknown program %q", name)
}

// ProgramForFenceLang maps a Markdown fence language tag to its canonical
// program name ("" = unknown). Exported for the fleet debug surface.
func ProgramForFenceLang(lang string) string {
	return fenceLangToProgram(lang)
}

// fenceLangToProgram maps a Markdown fence language tag to the canonical
// program name used for allowlist checks (python, bash, node).
// Returns "" for unrecognised tags.
func fenceLangToProgram(lang string) string {
	if interp, ok := currentRegistry().byLabel[strings.ToLower(lang)]; ok {
		return interp.Name
	}
	return ""
}

// FencedBlock is one fenced code block extracted from a Markdown body.
type FencedBlock struct {
	Lang string
	Code string
}

// ExtractFencedBlocks returns all fenced code blocks from body in document order.
func ExtractFencedBlocks(body string) []FencedBlock {
	var blocks []FencedBlock
	rest := body
	for {
		idx := strings.Index(rest, "```")
		if idx == -1 {
			break
		}
		after := rest[idx+3:]
		nl := strings.IndexByte(after, '\n')
		if nl == -1 {
			break
		}
		langStr := strings.TrimSpace(after[:nl])
		content := after[nl+1:]
		end := strings.Index(content, "```")
		if end == -1 {
			break
		}
		blocks = append(blocks, FencedBlock{Lang: langStr, Code: content[:end]})
		rest = content[end+3:]
	}
	return blocks
}

// extractFirstFencedBlock returns the fence language tag and code body of the
// first fenced code block (```lang\n...\n```) in body.
// Returns ("", "") when no complete block is found.
func extractFirstFencedBlock(body string) (string, string) {
	blocks := ExtractFencedBlocks(body)
	if len(blocks) == 0 {
		return "", ""
	}
	return blocks[0].Lang, blocks[0].Code
}

// buildChildEnv constructs the child process environment: a minimal base
// (PATH + FLEET_INPUT + the env declared in interpreters.json, globally and for
// this interpreter) plus selected parent vars via the explicit-allowlist
// (passthrough) and prefix-match mechanisms.
//
// Secret-scrub guarantee: only vars that are explicitly listed in passthrough
// OR whose name starts with an entry in prefix are forwarded. No other parent
// env var is included. An empty passthrough AND empty prefix → base only.
func buildChildEnv(inputFile string, interp *interpreter, passthrough, prefix []string) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"FLEET_INPUT=" + inputFile,
	}
	env = append(env, resolveEnvVars(currentRegistry().env)...)
	if interp != nil {
		env = append(env, resolveEnvVars(interp.Env)...)
	}
	if len(passthrough) == 0 && len(prefix) == 0 {
		return env
	}
	passthroughSet := make(map[string]struct{}, len(passthrough))
	for _, name := range passthrough {
		passthroughSet[name] = struct{}{}
	}
	for _, kv := range os.Environ() {
		name, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if _, found := passthroughSet[name]; found {
			env = append(env, kv)
			continue
		}
		for _, p := range prefix {
			if strings.HasPrefix(name, p) {
				env = append(env, kv)
				break
			}
		}
	}
	return env
}

// resolveEnvVars renders declared env entries, skipping any whose IfExists path
// is absent. The values are static config, not parent env, so the secret-scrub
// guarantee holds.
func resolveEnvVars(decls []envVar) []string {
	var env []string
	for _, v := range decls {
		if v.Name == "" {
			continue
		}
		if v.IfExists != "" {
			if _, err := os.Stat(v.IfExists); err != nil {
				continue
			}
		}
		env = append(env, v.Name+"="+v.Value)
	}
	return env
}

// limitedBuffer is an io.Writer that accumulates up to limit bytes and silently
// discards the rest. Used to bound stdout/stderr capture.
type limitedBuffer struct {
	limit int
	data  []byte
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	rem := b.limit - len(b.data)
	if rem > 0 {
		if len(p) <= rem {
			b.data = append(b.data, p...)
		} else {
			b.data = append(b.data, p[:rem]...)
		}
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string { return string(b.data) }
