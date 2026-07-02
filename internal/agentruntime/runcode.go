package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/webhookutil"
)

// CodeInput is the parameters for a code-role run (executor: code).
// Body must be pre-rendered (Jet-evaluated by the caller); RunCode extracts
// the first fenced code block from the rendered text.
type CodeInput struct {
	Body            string        // Jet-rendered role body
	Program         string        // optional override; empty → derived from fence language
	WritePatterns   []string      // scope enforcement (same semantics as LLM write_patterns)
	KB              KB            // write destination
	AllowedPrograms []string      // fleet allowlist; empty → code execution disabled
	Timeout         time.Duration // per-run timeout; 0 → bounded by ctx only
	Input           []byte        // delivery bag JSON written to $FLEET_INPUT
	EnvPassthrough  []string      // exact parent env var names forwarded to child
	EnvPrefix       []string      // parent env var name prefixes forwarded to child
	MaxStdoutBytes  int           // stdout cap per code child; 0 → 1 MiB default
	Sandbox         SandboxPolicy // OS-level isolation; zero value = safe default (native)
}

// RunCode executes a code role. It:
//  1. Extracts all fenced code blocks from Body (document order).
//  2. Resolves each block's program (explicit override wins, then fence lang)
//     and checks it against AllowedPrograms (empty list = disabled) up front,
//     so a disallowed later block fails before any block runs.
//  3. Runs the blocks as a pipeline via RunBlock (each in its own
//     secret-scrubbed env + throwaway workdir): block i's stdout is block
//     i+1's stdin, shell-pipe style. The first block gets no stdin; every
//     block sees the delivery bag via $FLEET_INPUT. A non-zero exit stops the
//     pipeline. Intermediate stdout is free-form.
//  4. Parses the LAST block's stdout as {"changes":[...],"answer":"..."} JSON.
//  5. Applies each change via ScopedKB(WritePatterns) — the same scope
//     enforcement as write_note. Out-of-scope paths are denied, not written.
//
// Returns *Result with TokensUsed=0, Steps=len(blocks) on success. exec/code
// is NOT a scope bypass: all writes go through write_patterns enforcement.
func RunCode(ctx context.Context, in CodeInput) (*Result, error) {
	if in.KB == nil {
		return nil, errors.New("coderun: KB is required")
	}

	blocks := ExtractFencedBlocks(in.Body)
	if len(blocks) == 0 {
		return nil, errors.New("coderun: no fenced code block found in rendered body")
	}

	// Resolve + allowlist-check every block before running any: no partial
	// side effects when a later block is misconfigured. The explicit Program
	// override applies to every block; mixed-language pipelines leave it unset
	// and label each fence.
	programs := make([]string, len(blocks))
	for i, b := range blocks {
		program := in.Program
		if program == "" {
			program = fenceLangToProgram(b.Lang)
		}
		if program == "" {
			return nil, fmt.Errorf("coderun: %sfence language %q not supported (use python, bash, or node)", blockPrefix(i, len(blocks)), b.Lang)
		}
		if !isAllowed(program, in.AllowedPrograms) {
			if len(in.AllowedPrograms) == 0 {
				return nil, errors.New("coderun: code execution disabled (set --allowed-programs)")
			}
			return nil, fmt.Errorf("coderun: %sprogram %q not in --allowed-programs %v", blockPrefix(i, len(blocks)), program, in.AllowedPrograms)
		}
		programs[i] = program
	}

	// Pipeline: in.Timeout bounds each block individually; ctx bounds the whole run.
	var stdin []byte
	var stdout string
	for i, b := range blocks {
		out, _, _, runErr := RunBlock(ctx, CodeSpec{
			Program:        programs[i],
			Code:           b.Code,
			Stdin:          stdin,
			Input:          in.Input,
			Timeout:        in.Timeout,
			EnvPassthrough: in.EnvPassthrough,
			EnvPrefix:      in.EnvPrefix,
			MaxStdoutBytes: in.MaxStdoutBytes,
			Sandbox:        in.Sandbox,
		})
		if runErr != nil {
			return nil, fmt.Errorf("coderun: %s%w", blockPrefix(i, len(blocks)), runErr)
		}
		stdout = out
		stdin = []byte(out)
	}

	rawChanges, answer, perr := parseCodeOutput(stdout)
	if perr != nil {
		return nil, perr
	}

	// Apply changes through ScopedKB — NOT a scope bypass. Every write goes
	// through write_patterns enforcement just like write_note.
	// No read scope: code roles read from the delivery bag ($FLEET_INPUT).
	scoped := NewScopedKB(in.KB, nil, in.WritePatterns)
	res := &Result{
		Status:     StatusCompleted,
		Steps:      len(blocks),
		TokensUsed: 0,
		Answer:     answer,
	}
	for _, ch := range rawChanges {
		var applyErr error
		switch ch.Kind {
		case webhookutil.AgentChangeKindPatch:
			applyErr = scoped.Patch(ctx, ch.Path, ch.Find, ch.Replace)
		default: // AgentChangeKindWrite or empty
			applyErr = scoped.Write(ctx, ch.Path, ch.Content)
		}
		if errors.Is(applyErr, ErrWriteDenied) {
			res.Denials = append(res.Denials, "write "+ch.Path)
			continue
		}
		if applyErr != nil {
			return nil, fmt.Errorf("coderun: apply %s: %w", ch.Path, applyErr)
		}
		res.Changes = append(res.Changes, ch)
	}
	return res, nil
}

// blockPrefix returns "block i/n: " for multi-block pipelines and "" for a
// single block, keeping single-block error messages unchanged.
func blockPrefix(i, n int) string {
	if n <= 1 {
		return ""
	}
	return fmt.Sprintf("block %d/%d: ", i+1, n)
}

// isAllowed reports whether program is in the allowedPrograms list.
// Both program and list entries use canonical names (python, bash, node).
func isAllowed(program string, allowedPrograms []string) bool {
	for _, p := range allowedPrograms {
		if p == program {
			return true
		}
	}
	return false
}
