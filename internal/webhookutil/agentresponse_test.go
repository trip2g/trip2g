package webhookutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAgentResponse_Empty(t *testing.T) {
	resp, err := ParseAgentResponse(nil)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestParseAgentResponse_InvalidJSON(t *testing.T) {
	resp, err := ParseAgentResponse([]byte("not json"))
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestParseAgentResponse_NoChanges(t *testing.T) {
	body := []byte(`{"status":"ok","message":"nothing to do"}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "ok", resp.Status)
	require.Empty(t, resp.Changes)
}

func TestParseAgentResponse_WithChanges(t *testing.T) {
	body := []byte(`{
		"status": "ok",
		"message": "fixed 1 file",
		"changes": [
			{"path": "blog/post.md", "content": "# Fixed"}
		]
	}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Changes, 1)
	require.Equal(t, "blog/post.md", resp.Changes[0].Path)
	require.Equal(t, "# Fixed", resp.Changes[0].Content)
}

func TestParseAgentResponse_MissingPath(t *testing.T) {
	body := []byte(`{"changes": [{"content": "no path"}]}`)
	_, err := ParseAgentResponse(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid change")
}

func TestParseAgentResponse_MissingContent(t *testing.T) {
	body := []byte(`{"changes": [{"path": "test.md"}]}`)
	_, err := ParseAgentResponse(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid change")
}

func TestParseAgentResponse_WithExpectedHash(t *testing.T) {
	hash := "abc123"
	body := []byte(`{
		"changes": [
			{"path": "test.md", "content": "# Test", "expected_hash": "abc123"}
		]
	}`)
	resp, err := ParseAgentResponse(body)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, &hash, resp.Changes[0].ExpectedHash)
}
