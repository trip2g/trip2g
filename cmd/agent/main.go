// Command agent is the Phase-1 micro-agent executor CLI. It runs an instruction
// through an LLM tool-call loop over a directory vault, scoped by read/write
// glob patterns, with a non-overridable per-run token hard-cap. The LLM endpoint
// is any OpenAI-compatible API (set -base-url for a local or vendor model).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"trip2g/internal/agentruntime"
	"trip2g/internal/webhookutil"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "agent: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		instruction     = flag.String("instruction", "", "instruction text (the role prompt)")
		instructionFile = flag.String("instruction-file", "", "read the instruction from this file")
		readPatterns    = flag.String("read-patterns", `["**"]`, "JSON array of read-scope glob patterns")
		writePatterns   = flag.String("write-patterns", `[]`, "JSON array of write-scope glob patterns")
		vault           = flag.String("vault", ".", "path to the vault directory")
		model           = flag.String("model", "gpt-4o-mini", "model name")
		baseURL         = flag.String("base-url", "", "OpenAI-compatible base URL (empty = default OpenAI)")
		apiKey          = flag.String("api-key", "", "API key (falls back to OPENAI_API_KEY env)")
		maxTokens       = flag.Int("max-tokens", 100000, "non-overridable per-run token hard-cap")
		maxSteps        = flag.Int("max-steps", 25, "max tool-loop iterations")
	)
	flag.Parse()

	instructionText := *instruction
	if *instructionFile != "" {
		data, readErr := os.ReadFile(*instructionFile)
		if readErr != nil {
			return fmt.Errorf("read instruction file: %w", readErr)
		}
		instructionText = string(data)
	}
	if instructionText == "" {
		return errors.New("an -instruction or -instruction-file is required")
	}

	readPats, err := webhookutil.ParseJSONStringArray(*readPatterns)
	if err != nil {
		return fmt.Errorf("parse read-patterns: %w", err)
	}
	writePats, err := webhookutil.ParseJSONStringArray(*writePatterns)
	if err != nil {
		return fmt.Errorf("parse write-patterns: %w", err)
	}

	key := *apiKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}

	llm := agentruntime.NewOpenAILLM(key, *baseURL)
	kb := agentruntime.NewFileKB(*vault)

	result, err := agentruntime.Run(context.Background(), agentruntime.Input{
		Instruction:   instructionText,
		ReadPatterns:  readPats,
		WritePatterns: writePats,
		Model:         *model,
		MaxTokens:     *maxTokens,
		MaxSteps:      *maxSteps,
		LLM:           llm,
		KB:            kb,
	})
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(out))
	return nil
}
