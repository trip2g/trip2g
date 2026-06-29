// Command llmmock is a deterministic OpenAI-compatible chat completions stub
// for e2e fleet tests. It returns a write_note tool call on the first LLM
// request for any agent run, then finish on subsequent calls. No external
// dependencies — stdlib net/http only.
//
// Endpoint: POST /v1/chat/completions (matches the path go-openai appends to
// a custom BaseURL).
// Health:   GET  /health → 200 OK
//
// Flags / env:
//
//	--listen / LLMMOCK_LISTEN : listen address (default ":9091")
//
// First-call semantics: if the request messages contain no "tool" role message,
// this is the first call in a run → return a write_note tool call that writes
// "processed: <last-user-message>" to segments/sample.md.
// Subsequent calls (tool result present) → return finish({answer:"done"}).
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	var listen string
	flag.StringVar(&listen, "listen", envOr("LLMMOCK_LISTEN", ":9091"), "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/v1/chat/completions", handleChat)

	srv := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("llmmock: listening on %s", listen)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("llmmock: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

// chatMessage mirrors the subset of OpenAI chat message fields we need.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest mirrors the subset of the OpenAI chat completions request we need.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("llmmock: /v1/chat/completions called from %s", r.RemoteAddr)
	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	var req chatRequest
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = "mock"
	}

	// Determine call phase by checking for any tool-role message.
	// First call in any run: no tool messages yet → write_note.
	// Subsequent calls: tool result(s) present → finish.
	hasToolResult := false
	// Extract the instruction from the system message so the written content
	// is unique per delivery (the system prompt embeds the full instruction,
	// which includes the transcript content and the per-run unique Run: id).
	instructionContent := "Begin."
	for _, m := range req.Messages {
		if m.Role == "tool" {
			hasToolResult = true
		}
		if m.Role == "system" {
			instructionContent = extractInstruction(m.Content)
		}
	}

	var toolCall map[string]any
	if hasToolResult {
		// Second+ call: end the run.
		toolCall = map[string]any{
			"id":   "t2",
			"type": "function",
			"function": map[string]any{
				"name":      "finish",
				"arguments": `{"answer":"done"}`,
			},
		}
	} else {
		// First call: write a note that the spec can assert.
		// instructionContent is derived from the system prompt and contains
		// the rendered instruction (including transcript content + unique Run: id),
		// so every delivery produces a distinct content hash, avoiding no-op writes.
		content := "processed: " + truncate(instructionContent, 200)
		args, _ := json.Marshal(map[string]string{
			"path":    "segments/sample.md",
			"content": content,
		})
		toolCall = map[string]any{
			"id":   "t1",
			"type": "function",
			"function": map[string]any{
				"name":      "write_note",
				"arguments": string(args),
			},
		}
	}

	resp := map[string]any{
		"id":     "cmpl-mock",
		"object": "chat.completion",
		"model":  model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":       "assistant",
					"content":    nil,
					"tool_calls": []map[string]any{toolCall},
				},
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     len(body) / 4,
			"completion_tokens": 5,
			"total_tokens":      len(body)/4 + 5,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("llmmock: encode response: %v", err)
	}
}

// extractInstruction pulls the rendered instruction out of the agent system
// prompt (between "Instruction:\n" and "\n\nAccess scope") and strips any leading
// YAML frontmatter, so the written note content is unique per delivery (it then
// carries the transcript body + the per-run Run: id).
func extractInstruction(systemContent string) string {
	const prefix = "\nInstruction:\n"
	const suffix = "\n\nAccess scope"
	start := strings.Index(systemContent, prefix)
	if start < 0 {
		return systemContent
	}
	rest := systemContent[start+len(prefix):]
	if end := strings.Index(rest, suffix); end >= 0 {
		rest = rest[:end]
	}
	if strings.HasPrefix(rest, "---\n") {
		if end2 := strings.Index(rest[4:], "\n---\n"); end2 >= 0 {
			rest = rest[4+end2+5:] // skip past the closing ---
		}
	}
	return strings.TrimSpace(rest)
}

func truncate(s string, n int) string {
	// Trim leading/trailing whitespace so "Begin.\n" → "Begin."
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
