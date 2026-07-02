package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"

	"trip2g/internal/fleet"
	"trip2g/internal/logger"
)

// newOpenAIStub starts an httptest server speaking the OpenAI chat-completions
// wire shape, returning the scripted responses in order (the last one repeats).
func newOpenAIStub(t *testing.T, responses []string) *httptest.Server {
	t.Helper()
	var call int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		idx := call
		if idx >= len(responses) {
			idx = len(responses) - 1
		}
		call++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, responses[idx])
	}))
	t.Cleanup(srv.Close)
	return srv
}

// toolCallCompletion renders one chat completion whose message is a single
// tool call with the given name and JSON arguments.
func toolCallCompletion(name string, args map[string]any) string {
	raw, _ := json.Marshal(args)
	return `{"id":"1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant",` +
		`"tool_calls":[{"id":"c1","type":"function","function":{"name":"` + name + `","arguments":` +
		strMustQuote(string(raw)) + `}}]},"finish_reason":"tool_calls"}],` +
		`"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
}

func strMustQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// TestRunOnce_EndToEnd is the --once end-to-end path: a role note runs offline
// against a file KB with a stub LLM (write_note then finish); the write must
// land in the vault and the run must complete without a trip2g connection.
func TestRunOnce_EndToEnd(t *testing.T) {
	vault := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(vault, "notes"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(vault, "notes", "input.md"), []byte("raw meeting notes"), 0o644))

	roleDir := t.TempDir()
	rolePath := filepath.Join(roleDir, "summarizer.md")
	role := `---
mode: change
trigger_on: [update]
trigger_include: [notes/**]
read_patterns: [notes/**]
write_patterns: [notes/**]
tools: [read_note, write_note]
model: test-model
---
Read notes/input.md and write a summary to notes/summary.md.
`
	require.NoError(t, os.WriteFile(rolePath, []byte(role), 0o644))

	llmStub := newOpenAIStub(t, []string{
		toolCallCompletion("write_note", map[string]any{"path": "notes/summary.md", "content": "SUMMARY"}),
		toolCallCompletion("finish", map[string]any{"answer": "summarized"}),
	})

	cli := cliFlags{
		cfg: fleet.Config{
			LLMAPIKey:    "test-key",
			LLMBaseURL:   llmStub.URL + "/v1",
			DefaultModel: "test-model",
			TokenCeiling: 100000,
			StepCeiling:  25,
			OfferedTools: []string{"search", "read_note", "write_note", "patch_note"},
		},
		oncePath: rolePath,
		vaultDir: vault,
	}

	require.NoError(t, runOnce(context.Background(), cli))

	written, err := os.ReadFile(filepath.Join(vault, "notes", "summary.md"))
	require.NoError(t, err)
	require.Equal(t, "SUMMARY", string(written), "--once must persist the agent's write to the file KB")
}

// TestRunDryRun_Integration drives runDryRun against a fake trip2g GraphQL
// endpoint: it must print each role's resolved config (flagging invalid ones)
// and issue ONLY the DiscoverRoles query — never a webhook mutation.
func TestRunDryRun_Integration(t *testing.T) {
	goodRole := "---\nmode: change\ntrigger_on: [update]\ntrigger_include: [boards/**]\nread_patterns: [boards/**]\nwrite_patterns: [boards/**]\ntools: [search, patch_note]\n---\nBody.\n"

	var ops []string
	gqlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
		}
		var in struct {
			OperationName string `json:"operationName"`
		}
		if uerr := json.Unmarshal(raw, &in); uerr != nil {
			t.Errorf("decode request: %v", uerr)
		}
		ops = append(ops, in.OperationName)

		w.Header().Set("Content-Type", "application/json")
		if in.OperationName != "DiscoverRoles" {
			fmt.Fprint(w, `{"errors":[{"message":"unexpected operation `+in.OperationName+`"}]}`)
			return
		}
		notes := []map[string]any{
			{
				"value":   "roles/good.md",
				"content": goodRole,
				"latestNoteView": map[string]any{"meta": []map[string]string{
					{"key": "mode", "raw": "change"},
					{"key": "trigger_on", "raw": "[update]"},
					{"key": "trigger_include", "raw": "[boards/**]"},
					{"key": "read_patterns", "raw": "[boards/**]"},
					{"key": "write_patterns", "raw": "[boards/**]"},
					{"key": "tools", "raw": "[search, patch_note]"},
				}},
			},
			{
				"value":          "roles/bad.md",
				"content":        "No trigger config.\n",
				"latestNoteView": map[string]any{"meta": []map[string]string{{"key": "mode", "raw": "change"}}},
			},
		}
		data, err := json.Marshal(map[string]any{"notePaths": notes})
		if err != nil {
			t.Errorf("marshal response: %v", err)
		}
		fmt.Fprint(w, `{"data":`+string(data)+`}`)
	}))
	t.Cleanup(gqlSrv.Close)

	cfg := fleet.Config{
		OfferedTools: []string{"search", "read_note", "patch_note", "write_note"},
		DefaultModel: "gpt-4o-mini",
		AgentsFolder: "roles/",
	}
	gql := graphql.NewClient(gqlSrv.URL, &http.Client{Timeout: 5 * time.Second})
	discovery := fleet.NewDiscovery(gql, cfg.AgentsFolder, cfg.OfferedTools)

	var out bytes.Buffer
	runDryRun(context.Background(), &logger.DummyLogger{}, discovery, cfg, &out)

	report := out.String()
	require.Contains(t, report, "roles/good.md")
	require.Contains(t, report, "STATUS: OK")
	require.Contains(t, report, "roles/bad.md")
	require.Contains(t, report, "FLAGGED")

	require.Equal(t, []string{"DiscoverRoles"}, ops,
		"--dry-run must only discover; webhook queries/mutations are forbidden")
	require.NotContains(t, report, "error", "report should be clean")
}
