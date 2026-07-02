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
//  1. Extracts the first fenced code block from Body.
//  2. Resolves the program (explicit override wins, then fence lang).
//  3. Checks the program against AllowedPrograms (empty list = disabled).
//  4. Runs the code via RunBlock (secret-scrubbed env, throwaway workdir).
//  5. Parses stdout as {"changes":[...],"answer":"..."} JSON.
//  6. Applies each change via ScopedKB(WritePatterns) — the same scope
//     enforcement as write_note. Out-of-scope paths are denied, not written.
//
// Returns *Result with TokensUsed=0, Steps=1 on success. exec/code is NOT a
// scope bypass: all writes go through write_patterns enforcement.
func RunCode(ctx context.Context, in CodeInput) (*Result, error) {
	if in.KB == nil {
		return nil, errors.New("coderun: KB is required")
	}

	// extractAllFencedBlocks collects all blocks in document order; v1 runs only
	// the first block. Future pipe support will run multiple blocks sequentially.
	blocks := extractAllFencedBlocks(in.Body)
	if len(blocks) == 0 {
		return nil, errors.New("coderun: no fenced code block found in rendered body")
	}
	lang := blocks[0].Lang
	code := blocks[0].Code

	program := in.Program
	if program == "" {
		program = fenceLangToProgram(lang)
	}
	if program == "" {
		return nil, fmt.Errorf("coderun: fence language %q not supported (use python, bash, or node)", lang)
	}

	if !isAllowed(program, in.AllowedPrograms) {
		if len(in.AllowedPrograms) == 0 {
			return nil, errors.New("coderun: code execution disabled (set --allowed-programs)")
		}
		return nil, fmt.Errorf("coderun: program %q not in --allowed-programs %v", program, in.AllowedPrograms)
	}

	stdout, _, _, runErr := RunBlock(ctx, CodeSpec{
		Program:        program,
		Code:           code,
		Input:          in.Input,
		Timeout:        in.Timeout,
		EnvPassthrough: in.EnvPassthrough,
		EnvPrefix:      in.EnvPrefix,
		MaxStdoutBytes: in.MaxStdoutBytes,
		Sandbox:        in.Sandbox,
	})
	if runErr != nil {
		return nil, fmt.Errorf("coderun: %w", runErr)
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
		Steps:      1,
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
