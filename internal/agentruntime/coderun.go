package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"trip2g/internal/webhookutil"
)

// Program name string constants used in programBinary, fenceLangToProgram, and
// codeExt to satisfy the goconst linter (each string appears 3+ times).
const (
	progPython     = "python"
	progPython3    = "python3"
	progBash       = "bash"
	progJavaScript = "javascript"
	progNode       = "node"
)

// maxStdoutBytes caps code-child stdout to prevent memory exhaustion.
const maxStdoutBytes = 1 << 20 // 1 MiB

// CodeSpec is the parameters for one code-execution call.
type CodeSpec struct {
	Program        string        // canonical program name or fence lang (python, bash, node, ...)
	Code           string        // source text written to a per-run temp file
	Input          []byte        // delivery bag JSON; nil or empty → "{}"
	Timeout        time.Duration // per-run timeout; 0 = bounded by ctx only
	EnvPassthrough []string      // exact parent env var names to forward to child
	EnvPrefix      []string      // parent env var name prefixes to forward to child
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
// stdout is capped at maxStdoutBytes; stderr is also captured for diagnostics.
// Non-zero exit or context timeout returns an error; timedOut distinguishes
// the two failure modes.
func RunBlock(ctx context.Context, spec CodeSpec) (string, string, bool, error) {
	binary, berr := programBinary(spec.Program)
	if berr != nil {
		return "", "", false, berr
	}

	// Per-run throwaway workdir: prevents cross-run contamination.
	workDir, mkErr := os.MkdirTemp("", "fleet-coderun-*")
	if mkErr != nil {
		return "", "", false, fmt.Errorf("coderun: create workdir: %w", mkErr)
	}
	defer os.RemoveAll(workDir)

	codeFile := filepath.Join(workDir, "run"+codeExt(spec.Program))
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

	cmd := exec.CommandContext(runCtx, binary, codeFile)
	cmd.Dir = workDir
	// Secret-scrub: build an explicit MINIMAL env. NEVER nil (inherits parent)
	// or os.Environ() (exposes all secrets). The base is PATH + FLEET_INPUT only.
	// Selective pass-through adds only explicitly declared vars/prefixes on top.
	cmd.Env = buildChildEnv(inputFile, spec.EnvPassthrough, spec.EnvPrefix)

	var outBuf, errBuf limitedBuffer
	outBuf.limit = maxStdoutBytes
	errBuf.limit = maxStdoutBytes
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

// programBinary resolves a program name or fence language tag to the binary to
// run. Returns an error for unrecognized names so the caller can fail fast.
func programBinary(name string) (string, error) {
	switch name {
	case progPython, "py", progPython3:
		return progPython3, nil
	case progBash, "sh":
		return progBash, nil
	case "js", progJavaScript, progNode:
		return progNode, nil
	}
	return "", fmt.Errorf("coderun: unknown program %q (supported: python, bash, node)", name)
}

// fenceLangToProgram maps a Markdown fence language tag to the canonical
// program name used for allowlist checks (python, bash, node).
// Returns "" for unrecognised tags.
func fenceLangToProgram(lang string) string {
	switch strings.ToLower(lang) {
	case progPython, "py":
		return progPython
	case progBash, "sh":
		return progBash
	case "js", progJavaScript, progNode:
		return progNode
	}
	return ""
}

// extractFirstFencedBlock returns the fence language tag and code body of the
// first fenced code block (```lang\n...\n```) in body.
// Returns ("", "") when no complete block is found.
func extractFirstFencedBlock(body string) (string, string) {
	idx := strings.Index(body, "```")
	if idx == -1 {
		return "", ""
	}
	rest := body[idx+3:]
	nl := strings.IndexByte(rest, '\n')
	if nl == -1 {
		return "", ""
	}
	langStr := strings.TrimSpace(rest[:nl])
	content := rest[nl+1:]
	end := strings.Index(content, "```")
	if end == -1 {
		return "", ""
	}
	return langStr, content[:end]
}

// codeExt returns a filename extension for the program so runtimes that check
// the filename (e.g. node) work correctly.
func codeExt(program string) string {
	switch program {
	case progPython, "py", progPython3:
		return ".py"
	case "js", progJavaScript, progNode:
		return ".js"
	default: // bash, sh, unknown
		return ".sh"
	}
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
