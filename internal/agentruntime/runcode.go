package agentruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/coderun"
	"trip2g/internal/webhookutil"
)

// CodeInput is the parameters for a code-role run (executor: code). Body must be
// pre-rendered (Jet-evaluated by the caller); RunCode extracts and runs the
// fenced code blocks via the coderun package, then applies the returned changes
// through ScopedKB. The execution fields (Body/Program/AllowedPrograms/...) are
// forwarded to coderun.CodeInput; KB and WritePatterns stay here — the apply/
// scope half lives with fleet, not with the (KB-less) code executor.
type CodeInput struct {
	Body            string                // Jet-rendered role body
	Program         string                // optional override; empty → derived from fence language
	WritePatterns   []string              // scope enforcement (same semantics as LLM write_patterns)
	KB              KB                    // write destination
	AllowedPrograms []string              // fleet allowlist; empty → code execution disabled
	Timeout         time.Duration         // per-run timeout; 0 → bounded by ctx only
	Input           []byte                // delivery bag JSON written to $FLEET_INPUT
	EnvPassthrough  []string              // exact parent env var names forwarded to child
	EnvPrefix       []string              // parent env var name prefixes forwarded to child
	MaxStdoutBytes  int                   // stdout cap per code child; 0 → 1 MiB default
	Sandbox         coderun.SandboxPolicy // OS-level isolation; zero value = safe default (native)
}

// RunCode executes a code role. It:
//  1. Runs the fenced code blocks via coderun.Exec (single block or streaming
//     pipeline; program allowlist checked up front).
//  2. Parses the LAST block's stdout as {"changes":[...],"answer":"..."} JSON.
//  3. Applies each change via ScopedKB(WritePatterns) — the same scope
//     enforcement as write_note. Out-of-scope paths are denied, not written.
//
// Returns *Result with TokensUsed=0, Steps=len(blocks) on success. exec/code
// is NOT a scope bypass: all writes go through write_patterns enforcement.
func RunCode(ctx context.Context, in CodeInput) (*Result, error) {
	if in.KB == nil {
		return nil, errors.New("coderun: KB is required")
	}

	core, err := coderun.Exec(ctx, coderun.CodeInput{
		Body:            in.Body,
		Program:         in.Program,
		AllowedPrograms: in.AllowedPrograms,
		Timeout:         in.Timeout,
		Input:           in.Input,
		EnvPassthrough:  in.EnvPassthrough,
		EnvPrefix:       in.EnvPrefix,
		MaxStdoutBytes:  in.MaxStdoutBytes,
		Sandbox:         in.Sandbox,
	}, false)
	if err != nil {
		return nil, err
	}

	// Apply changes through ScopedKB — NOT a scope bypass. Every write goes
	// through write_patterns enforcement just like write_note. No read scope:
	// code roles read from the delivery bag ($FLEET_INPUT).
	scoped := NewScopedKB(in.KB, nil, in.WritePatterns)
	res := &Result{
		Status:     StatusCompleted,
		Steps:      core.Steps,
		TokensUsed: 0,
		Answer:     core.Answer,
	}
	for _, ch := range core.Changes {
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
