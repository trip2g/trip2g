package agentruntime

import (
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

// interpreter describes one language runtime: binary argv prefix, fence labels, temp-file extension.
type interpreter struct {
	Name            string   `json:"name"`
	Cmd             []string `json:"cmd"`
	CodeBlockLabels []string `json:"code_block_labels"`
	Ext             string   `json:"ext"`
}

// interpreterRegistry holds two indices built from the loaded interpreter list.
type interpreterRegistry struct {
	byLabel map[string]*interpreter
	byName  map[string]*interpreter
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
		panic("agentruntime: failed to build interpreter registry: " + err.Error())
	}
	return r
}

// buildRegistry parses an interpreters JSON blob and builds byLabel and byName indices.
func buildRegistry(data []byte) (*interpreterRegistry, error) {
	var payload struct {
		Interpreters []interpreter `json:"interpreters"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse interpreters JSON: %w", err)
	}
	reg := &interpreterRegistry{
		byLabel: make(map[string]*interpreter),
		byName:  make(map[string]*interpreter),
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
	Program        string        // canonical program name (python, bash, node, ...)
	Code           string        // source text written to a per-run temp file
	Input          []byte        // delivery bag JSON; nil or empty → "{}"
	Timeout        time.Duration // per-run timeout; 0 = bounded by ctx only
	EnvPassthrough []string      // exact parent env var names to forward to child
	EnvPrefix      []string      // parent env var name prefixes to forward to child
	MaxStdoutBytes int           // stdout cap; 0 → 1 MiB default
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
	cmd := exec.CommandContext(runCtx, interp.Cmd[0], append(interp.Cmd[1:], codeFile)...)
	cmd.Dir = workDir
	// Secret-scrub: build an explicit MINIMAL env. NEVER nil (inherits parent)
	// or os.Environ() (exposes all secrets). The base is PATH + FLEET_INPUT only.
	// Selective pass-through adds only explicitly declared vars/prefixes on top.
	cmd.Env = buildChildEnv(inputFile, spec.EnvPassthrough, spec.EnvPrefix)

	limit := spec.MaxStdoutBytes
	if limit == 0 {
		limit = 1 << 20
	}
	var outBuf, errBuf limitedBuffer
	outBuf.limit = limit
	errBuf.limit = limit
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
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

// parseCodeOutput parses the code child's stdout JSON into []AgentChange +
// answer string. stdout must match {"changes":[...],"answer":"..."}.
// A "content"-only entry → AgentChangeKindWrite; "find"/"replace" → AgentChangeKindPatch.
func parseCodeOutput(stdout string) ([]webhookutil.AgentChange, string, error) {
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

// fenceLangToProgram maps a Markdown fence language tag to the canonical
// program name used for allowlist checks (python, bash, node).
// Returns "" for unrecognised tags.
func fenceLangToProgram(lang string) string {
	if interp, ok := currentRegistry().byLabel[strings.ToLower(lang)]; ok {
		return interp.Name
	}
	return ""
}

// fencedBlock is one fenced code block extracted from a Markdown body.
type fencedBlock struct {
	Lang string
	Code string
}

// extractAllFencedBlocks returns all fenced code blocks from body in document order.
func extractAllFencedBlocks(body string) []fencedBlock {
	var blocks []fencedBlock
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
		blocks = append(blocks, fencedBlock{Lang: langStr, Code: content[:end]})
		rest = content[end+3:]
	}
	return blocks
}

// extractFirstFencedBlock returns the fence language tag and code body of the
// first fenced code block (```lang\n...\n```) in body.
// Returns ("", "") when no complete block is found.
func extractFirstFencedBlock(body string) (string, string) {
	blocks := extractAllFencedBlocks(body)
	if len(blocks) == 0 {
		return "", ""
	}
	return blocks[0].Lang, blocks[0].Code
}

// buildChildEnv constructs the child process environment: a minimal base
// (PATH + FLEET_INPUT) plus selected parent vars via the explicit-allowlist
// (passthrough) and prefix-match mechanisms.
//
// Secret-scrub guarantee: only vars that are explicitly listed in passthrough
// OR whose name starts with an entry in prefix are forwarded. No other parent
// env var is included. An empty passthrough AND empty prefix → base only.
func buildChildEnv(inputFile string, passthrough, prefix []string) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"FLEET_INPUT=" + inputFile,
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
