package codellm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"

	"trip2g/cmd/codellm/internal/coderun"
)

func TestChatCompletions_StdoutLimit(t *testing.T) {
	for _, pipeline := range []bool{false, true} {
		for _, script := range []string{
			`printf '{"answer":"too long"}'`,
			// A complete JSON prefix must not turn discarded bytes into success.
			`printf '{"answer":"ok"}     '`,
		} {
			t.Run(fmt.Sprintf("pipeline=%t/%s", pipeline, script), func(t *testing.T) {
				srv := New(Config{
					AllowedPrograms: []string{"bash"}, MaxStdoutBytes: 15,
					Sandbox: coderun.SandboxPolicy{Mode: coderun.SandboxOff},
				})
				msg := bashBody(script)
				if pipeline {
					msg.Content += "\n```bash\ncat\n```"
				}
				rec := doChat(t, srv, []goopenai.ChatCompletionMessage{msg})
				require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
				var got struct {
					Error struct {
						Type    string `json:"type"`
						Message string `json:"message"`
					} `json:"error"`
					Choices []any `json:"choices"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, "code_execution_error", got.Error.Type)
				require.Contains(t, got.Error.Message, "stdout limit exceeded (15 bytes)")
				require.Empty(t, got.Choices)
			})
		}
	}
}

func TestGraphQLRunBlocks_StdoutLimit(t *testing.T) {
	for _, pipeline := range []bool{false, true} {
		for _, size := range []int{10, 11} {
			t.Run(fmt.Sprintf("pipeline=%t/bytes=%d", pipeline, size), func(t *testing.T) {
				srv := New(Config{
					AllowedPrograms: []string{"bash"}, MaxStdoutBytes: 10,
					Sandbox: coderun.SandboxPolicy{Mode: coderun.SandboxOff},
				})
				blocks := fmt.Sprintf(`{kind: CODE, language: "bash", content: "head -c %d /dev/zero | tr '\\0' 'x'"}`, size)
				if pipeline {
					blocks += `, {kind: CODE, language: "bash", content: "cat"}`
				}
				query := fmt.Sprintf(`mutation { runBlocks(input: {
					input: {changedFiles: [], attachedNotes: [], depth: 1}, blocks: [%s]
				}) {
					__typename
					... on RunBlocksPayload {output}
					... on BlockErrorPayload {index message}
				} }`, blocks)
				body, err := json.Marshal(map[string]string{"query": query})
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, graphqlPrefix, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(rec, req)
				require.Equal(t, http.StatusOK, rec.Code)
				var got struct {
					Data struct {
						Run struct {
							Type    string `json:"__typename"`
							Output  string `json:"output"`
							Index   int    `json:"index"`
							Message string `json:"message"`
						} `json:"runBlocks"`
					} `json:"data"`
					Errors []any `json:"errors"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Empty(t, got.Errors)
				if size > 10 {
					require.Equal(t, "BlockErrorPayload", got.Data.Run.Type)
					require.Contains(t, got.Data.Run.Message, "stdout limit exceeded (10 bytes)")
					require.Empty(t, got.Data.Run.Output)
					index := 0
					if pipeline {
						index = 1
					}
					require.Equal(t, index, got.Data.Run.Index)
				} else {
					require.Equal(t, "RunBlocksPayload", got.Data.Run.Type)
					require.Equal(t, "xxxxxxxxxx", got.Data.Run.Output)
				}
			})
		}
	}
}
