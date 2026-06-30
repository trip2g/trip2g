package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/logger"
)

// TestHandlerStatus201 verifies that a config returning {status:201, headers:{"X-T":"v"}, body:{ok:true}}
// produces the expected status, header, and JSON body.
func TestHandlerStatus201(t *testing.T) {
	const cfg = `{status: 201, headers: {"X-T": "v"}, body: {ok: true}}`
	h := newHandler(cfg, &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, 201, w.Code)
	require.Equal(t, "v", w.Header().Get("X-T"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, true, body["ok"])
}

// TestHandlerBodyText verifies that a config returning {bodyText:"hi"} writes raw text.
func TestHandlerBodyText(t *testing.T) {
	const cfg = `{bodyText: "hi"}`
	h := newHandler(cfg, &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "hi", w.Body.String())
}

// TestHandlerDefaultContentType verifies that the default Content-Type is application/json
// when the config does not specify headers.
func TestHandlerDefaultContentType(t *testing.T) {
	const cfg = `{body: {x: 1}}`
	h := newHandler(cfg, &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

// TestHandlerEvalError verifies that a broken jsonnet config returns 500.
func TestHandlerEvalError(t *testing.T) {
	const cfg = `this is not valid jsonnet !!!`
	h := newHandler(cfg, &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func loadKrispConfig(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("configs/krisp.jsonnet")
	require.NoError(t, err, "load krisp.jsonnet")
	return string(src)
}

// TestKrispHealth verifies GET /health returns "ok".
func TestKrispHealth(t *testing.T) {
	h := newHandler(loadKrispConfig(t), &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "ok", w.Body.String())
}

// krisListResponse mirrors the top-level shape of POST /v2/meetings/list.
type krisListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Rows  []krisMeeting `json:"rows"`
		Count int           `json:"count"`
	} `json:"data"`
	ReqID string `json:"req_id"`
}

type krisMeeting struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Speakers []krisSpeaker `json:"speakers"`
}

type krisSpeaker struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// TestKrispMeetingsListPage1 verifies that page 1 returns 3 meetings with
// the expected IDs, names, and speakers — the contract the e2e ingest depends on.
func TestKrispMeetingsListPage1(t *testing.T) {
	h := newHandler(loadKrispConfig(t), &logger.DummyLogger{})

	body := strings.NewReader(`{"sort":"desc","sortKey":"created_at","page":1,"limit":200}`)
	req := httptest.NewRequest(http.MethodPost, "/v2/meetings/list", body)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp krisListResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "success", resp.Message)
	require.Equal(t, 3, resp.Data.Count)
	require.Len(t, resp.Data.Rows, 3)

	// Meeting 1.
	require.Equal(t, "aabbccddeeff00112233445566778800", resp.Data.Rows[0].ID)
	require.Equal(t, "Team Sync Q1 Planning", resp.Data.Rows[0].Name)
	require.Len(t, resp.Data.Rows[0].Speakers, 2)
	require.Equal(t, "Alice", resp.Data.Rows[0].Speakers[0].FirstName)
	require.Equal(t, "Mock", resp.Data.Rows[0].Speakers[0].LastName)
	require.Equal(t, "Alice@mock.example", resp.Data.Rows[0].Speakers[0].Email)
	require.Equal(t, "Bob", resp.Data.Rows[0].Speakers[1].FirstName)

	// Meeting 2.
	require.Equal(t, "1122334455667788aabbccddeeff0011", resp.Data.Rows[1].ID)
	require.Equal(t, "Product Demo and Feedback Session", resp.Data.Rows[1].Name)
	require.Equal(t, "Carol", resp.Data.Rows[1].Speakers[0].FirstName)
	require.Equal(t, "Dave", resp.Data.Rows[1].Speakers[1].FirstName)

	// Meeting 3.
	require.Equal(t, "99aabbccddeeff001122334455667788", resp.Data.Rows[2].ID)
	require.Equal(t, "Engineering Architecture Review", resp.Data.Rows[2].Name)
	require.Len(t, resp.Data.Rows[2].Speakers, 3)
	require.Equal(t, "Eve", resp.Data.Rows[2].Speakers[0].FirstName)
	require.Equal(t, "Frank", resp.Data.Rows[2].Speakers[1].FirstName)
	require.Equal(t, "Grace", resp.Data.Rows[2].Speakers[2].FirstName)
}

// TestKrispMeetingsListPage2Empty verifies that page > 1 returns empty rows
// while count still reflects the total.
func TestKrispMeetingsListPage2Empty(t *testing.T) {
	h := newHandler(loadKrispConfig(t), &logger.DummyLogger{})

	body := strings.NewReader(`{"page":2,"limit":200}`)
	req := httptest.NewRequest(http.MethodPost, "/v2/meetings/list", body)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp krisListResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Empty(t, resp.Data.Rows, "page 2 must return no rows")
	require.Equal(t, 3, resp.Data.Count)
}

// krissTreeChild is a minimal view of a block-tree child node.
type krissTreeChild struct {
	BlockType    string `json:"block_type"`
	SpeakerIndex int    `json:"speakerIndex"`
	Speech       struct {
		Text  string  `json:"text"`
		Start float64 `json:"start"`
	} `json:"speech"`
}

// krissTreeRoot is the top-level block tree response.
type krissTreeRoot struct {
	ID        string           `json:"id"`
	BlockType string           `json:"block_type"`
	Children  []krissTreeChild `json:"children"`
}

// TestKrispBlockTreeID1 verifies the block tree for meeting 1 has the expected
// utterances — first text, speakerIndex ordering, and block_type hierarchy.
func TestKrispBlockTreeID1(t *testing.T) {
	const id1 = "aabbccddeeff00112233445566778800"
	h := newHandler(loadKrispConfig(t), &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/v2/block/"+id1+"/tree", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var tree krissTreeRoot
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tree))

	require.Equal(t, id1, tree.ID)
	require.Equal(t, "meeting", tree.BlockType)
	require.Len(t, tree.Children, 4)

	// First utterance: speakerIndex 1, exact text.
	require.Equal(t, "utterance", tree.Children[0].BlockType)
	require.Equal(t, 1, tree.Children[0].SpeakerIndex)
	require.Equal(t, "Good morning, let us get started with the Q1 planning.", tree.Children[0].Speech.Text)
	require.InDelta(t, 0.0, tree.Children[0].Speech.Start, 0.001)

	// Second utterance: speakerIndex 2.
	require.Equal(t, 2, tree.Children[1].SpeakerIndex)
	require.Equal(t, "I have prepared the roadmap items for this quarter.", tree.Children[1].Speech.Text)
	require.InDelta(t, 15.5, tree.Children[1].Speech.Start, 0.001)
}

// TestKrispBlockTreeID2 verifies the block tree for meeting 2.
func TestKrispBlockTreeID2(t *testing.T) {
	const id2 = "1122334455667788aabbccddeeff0011"
	h := newHandler(loadKrispConfig(t), &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/v2/block/"+id2+"/tree", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var tree krissTreeRoot
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tree))

	require.Equal(t, id2, tree.ID)
	require.Len(t, tree.Children, 4)
	require.Equal(t, "Welcome everyone to the product demo session.", tree.Children[0].Speech.Text)
}

// TestKrispBlockTreeID3 verifies the block tree for meeting 3 has 5 utterances.
func TestKrispBlockTreeID3(t *testing.T) {
	const id3 = "99aabbccddeeff001122334455667788"
	h := newHandler(loadKrispConfig(t), &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/v2/block/"+id3+"/tree", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var tree krissTreeRoot
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tree))

	require.Len(t, tree.Children, 5)
	require.Equal(t, 3, tree.Children[2].SpeakerIndex)
	require.Equal(t, "The caching strategy needs to address cache invalidation.", tree.Children[2].Speech.Text)
}

// TestKrispBlockTreeDefaultID verifies that an unknown id returns 2 generic utterances.
func TestKrispBlockTreeDefaultID(t *testing.T) {
	const unknownID = "000000000000000000000000ffffffff"
	h := newHandler(loadKrispConfig(t), &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/v2/block/"+unknownID+"/tree", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var tree krissTreeRoot
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tree))

	require.Len(t, tree.Children, 2)
	require.Contains(t, tree.Children[0].Speech.Text, unknownID)
}

func loadLLMConfig(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("configs/llm.jsonnet")
	require.NoError(t, err, "load llm.jsonnet")
	return string(src)
}

// TestLLMHealth verifies GET /health returns "ok".
func TestLLMHealth(t *testing.T) {
	h := newHandler(loadLLMConfig(t), &logger.DummyLogger{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "ok", w.Body.String())
}

// llmResponse mirrors the top-level shape of POST /v1/chat/completions.
type llmResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string `json:"role"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// chatMessage is a minimal OpenAI chat message for building test request bodies.
type chatMessage struct {
	Role    string  `json:"role"`
	Content *string `json:"content"`
}

// chatRequest is a minimal OpenAI chat completions request for building test bodies.
type chatRequest struct {
	Model    string        `json:"model,omitempty"`
	Messages []chatMessage `json:"messages"`
}

// strPtr is a test helper that returns a pointer to s.
func strPtr(s string) *string { return &s }

// systemPromptWith builds a system message that embeds an instruction via the
// "\nInstruction:\n---\n<fm>\n---\n<body>\n\nAccess scope:" sentinel format.
func systemPromptWith(instruction string) string {
	return "System preamble.\nInstruction:\n---\nfm\n---\n" + instruction + "\n\nAccess scope: all"
}

// marshalBody marshals v to a JSON string for use as a request body.
func marshalBody(t *testing.T, v any) *strings.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return strings.NewReader(string(b))
}

// TestLLMFirstCall verifies the first call (no tool result) returns write_note
// with the extracted instruction as the content.
func TestLLMFirstCall(t *testing.T) {
	h := newHandler(loadLLMConfig(t), &logger.DummyLogger{})

	sysContent := systemPromptWith("Do the thing")
	body := marshalBody(t, chatRequest{
		Model: "gpt-4o",
		Messages: []chatMessage{
			{Role: "system", Content: strPtr(sysContent)},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp llmResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, "cmpl-mock", resp.ID)
	require.Equal(t, "chat.completion", resp.Object)
	require.Equal(t, "gpt-4o", resp.Model)
	require.Len(t, resp.Choices, 1)
	require.Equal(t, "tool_calls", resp.Choices[0].FinishReason)
	require.Equal(t, "assistant", resp.Choices[0].Message.Role)

	calls := resp.Choices[0].Message.ToolCalls
	require.Len(t, calls, 1)
	require.Equal(t, "t1", calls[0].ID)
	require.Equal(t, "write_note", calls[0].Function.Name)

	// arguments must be a JSON string containing {path, content}.
	var args map[string]string
	require.NoError(t, json.Unmarshal([]byte(calls[0].Function.Arguments), &args))
	require.Equal(t, "segments/sample.md", args["path"])
	require.Equal(t, "processed: Do the thing", args["content"])
}

// TestLLMSubsequentCall verifies that a message list containing a tool-role
// message triggers the finish tool call.
func TestLLMSubsequentCall(t *testing.T) {
	h := newHandler(loadLLMConfig(t), &logger.DummyLogger{})

	sysContent := systemPromptWith("Do the thing")
	toolContent := "ok"
	body := marshalBody(t, chatRequest{
		Model: "gpt-4o",
		Messages: []chatMessage{
			{Role: "system", Content: strPtr(sysContent)},
			{Role: "assistant", Content: nil},
			{Role: "tool", Content: strPtr(toolContent)},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp llmResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Choices, 1)
	require.Equal(t, "tool_calls", resp.Choices[0].FinishReason)

	calls := resp.Choices[0].Message.ToolCalls
	require.Len(t, calls, 1)
	require.Equal(t, "t2", calls[0].ID)
	require.Equal(t, "finish", calls[0].Function.Name)

	var args map[string]string
	require.NoError(t, json.Unmarshal([]byte(calls[0].Function.Arguments), &args))
	require.Equal(t, "done", args["answer"])
}

// TestLLMModelEchoed verifies the model field is echoed from the request.
func TestLLMModelEchoed(t *testing.T) {
	h := newHandler(loadLLMConfig(t), &logger.DummyLogger{})

	body := marshalBody(t, chatRequest{
		Model:    "my-custom-model",
		Messages: []chatMessage{{Role: "user", Content: strPtr("hello")}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp llmResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, "my-custom-model", resp.Model)
}

// TestLLMModelDefault verifies that an absent model field defaults to "mock".
func TestLLMModelDefault(t *testing.T) {
	h := newHandler(loadLLMConfig(t), &logger.DummyLogger{})

	body := marshalBody(t, chatRequest{
		Messages: []chatMessage{{Role: "user", Content: strPtr("hello")}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp llmResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, "mock", resp.Model)
}

// TestLLMInstructionNoPrefix verifies that when the system content has no
// "\nInstruction:\n" prefix, the whole content is used as the instruction.
func TestLLMInstructionNoPrefix(t *testing.T) {
	h := newHandler(loadLLMConfig(t), &logger.DummyLogger{})

	body := marshalBody(t, chatRequest{
		Model:    "m",
		Messages: []chatMessage{{Role: "system", Content: strPtr("Just do it")}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	w := httptest.NewRecorder()
	h(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp llmResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	calls := resp.Choices[0].Message.ToolCalls
	var args map[string]string
	require.NoError(t, json.Unmarshal([]byte(calls[0].Function.Arguments), &args))
	require.Equal(t, "processed: Just do it", args["content"])
}
