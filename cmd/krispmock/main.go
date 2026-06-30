// Command krispmock is a deterministic Krisp API stub for fleet e2e tests.
// It serves synthetic meetings and block-tree transcripts in the exact
// response shape of the real Krisp API, with no real user data.
//
// Endpoints:
//
//	POST /v2/meetings/list   → paginated list of synthetic meetings
//	GET  /v2/block/{id}/tree → block tree with embedded transcript utterances
//	GET  /health             → 200 "ok"
//
// Flags / env:
//
//	--listen / KRISPMOCK_LISTEN : listen address (default ":9092")
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	defaultListen    = ":9092"
	contentTypeJSON  = "application/json"
	bodyLimit        = 1 << 20 // 1 MiB.
	mockReqID        = "00000000-0000-0000-0000-000000000001"
	permEdit         = "edit"
	blockTypeMeeting = "meeting"
	blockTypeUtt     = "utterance"
	statusDone       = "done"
	meetingID1       = "aabbccddeeff00112233445566778800"
	meetingID2       = "1122334455667788aabbccddeeff0011"
	meetingID3       = "99aabbccddeeff001122334455667788"
)

// utteranceData holds one transcript segment for building synthetic block trees.
type utteranceData struct {
	speakerIdx int
	start      float64
	text       string
}

// listBody is the request body for POST /v2/meetings/list.
type listBody struct {
	Sort    string `json:"sort"`
	SortKey string `json:"sortKey"`
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
	IsOwner bool   `json:"isOwner"`
}

func main() {
	var listen string
	flag.StringVar(&listen, "listen", envOr("KRISPMOCK_LISTEN", defaultListen), "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /v2/meetings/list", handleMeetingsList)
	mux.HandleFunc("GET /v2/block/{id}/tree", handleBlockTree)

	srv := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("krispmock: listening on %s", listen)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("krispmock: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
}

func handleMeetingsList(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, bodyLimit))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	var req listBody
	if len(body) > 0 {
		if err = json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	meetings := syntheticMeetings()
	rows := meetings
	if req.Page > 1 {
		rows = []any{}
	}
	resp := map[string]any{
		"code":    0,
		"message": "success",
		"data": map[string]any{
			"rows":  rows,
			"count": len(meetings),
		},
		"req_id": mockReqID,
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("krispmock: encode meetings list: %v", err)
	}
}

func handleBlockTree(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Set("Content-Type", contentTypeJSON)
	if err := json.NewEncoder(w).Encode(syntheticTree(id)); err != nil {
		log.Printf("krispmock: encode block tree: %v", err)
	}
}

func syntheticMeetings() []any {
	return []any{
		makeMeeting(meetingID1, "Team Sync Q1 Planning",
			"2026-01-15T10:00:00Z", 3600, []string{"Alice", "Bob"}),
		makeMeeting(meetingID2, "Product Demo and Feedback Session",
			"2026-02-20T14:30:00Z", 2700, []string{"Carol", "Dave"}),
		makeMeeting(meetingID3, "Engineering Architecture Review",
			"2026-03-10T09:00:00Z", 5400, []string{"Eve", "Frank", "Grace"}),
	}
}

func makeMeeting(id, name, startedAt string, duration int, names []string) map[string]any {
	return map[string]any{
		"id":                          id,
		"name":                        name,
		"started_at":                  startedAt,
		"duration":                    duration,
		"speakers":                    makeSpeakers(names),
		"is_demo":                     false,
		"created_at":                  startedAt,
		"includes_external_attendees": false,
		"resources": map[string]any{
			"transcript": map[string]any{
				"status":    "complete",
				"processor": "krisp",
			},
			"recording":     false,
			"recordings":    []any{},
			"meeting_notes": map[string]any{},
		},
		"thumbnails": []any{},
		"user_interactions": map[string]any{
			"read":         false,
			"starred":      false,
			"hidden":       false,
			"listen_later": false,
			"progress":     nil,
			"is_new":       true,
		},
		"app_name":  "Meet",
		"parent_id": nil,
		"accesses":  []any{},
		"status":    statusDone,
		"self_access": map[string]any{
			"is_owner": true,
			"type":     "personal",
			"resources": map[string]any{
				"transcript":   permEdit,
				"meeting_note": permEdit,
				"recording":    permEdit,
				"agenda":       permEdit,
			},
		},
		"is_private":   false,
		"highlight":    nil,
		"tags":         []any{},
		"folders":      []any{},
		"storage_size": 0,
	}
}

func makeSpeakers(names []string) []any {
	speakers := make([]any, len(names))
	for i, firstName := range names {
		speakers[i] = map[string]any{
			"id":         speakerID(i + 1),
			"first_name": firstName,
			"last_name":  "Mock",
			"email":      firstName + "@mock.example",
			"photo":      "",
		}
	}
	return speakers
}

func speakerID(n int) string {
	return fmt.Sprintf("%032x", n)
}

func syntheticTree(id string) map[string]any {
	utterances := treeUtterances(id)
	return map[string]any{
		"id":            id,
		"permission":    permEdit,
		"label":         "label",
		"block_type":    blockTypeMeeting,
		"resources":     []any{},
		"children":      buildBlockChildren(id, utterances),
		"associated_to": []any{},
		"$version":      1,
		"content": map[string]any{
			"is_demo": false,
			"status":  statusDone,
		},
		"self_access": map[string]any{
			"is_owner": true,
			"type":     "personal",
			"resources": map[string]any{
				"transcript":   permEdit,
				"meeting_note": permEdit,
				"recording":    permEdit,
				"agenda":       permEdit,
			},
		},
		"accesses":        map[string]any{},
		"access_requests": map[string]any{},
		"folders":         []any{},
	}
}

func buildBlockChildren(meetingID string, utterances []utteranceData) []any {
	children := make([]any, len(utterances))
	for i, u := range utterances {
		children[i] = map[string]any{
			"id":           blockChildID(meetingID, i),
			"permission":   permEdit,
			"label":        "label",
			"block_type":   blockTypeUtt,
			"speakerIndex": u.speakerIdx,
			"speech": map[string]any{
				"text":  u.text,
				"start": u.start,
			},
			"resources":     []any{},
			"children":      []any{},
			"associated_to": []any{},
			"$version":      1,
			"content": map[string]any{
				"is_edited": false,
				"status":    "ready",
				"language":  "en-US",
			},
		}
	}
	return children
}

func blockChildID(meetingID string, pos int) string {
	const minIDLen = 24
	if len(meetingID) < minIDLen {
		return fmt.Sprintf("%032x", pos)
	}
	return fmt.Sprintf("%s%08x", meetingID[:minIDLen], pos)
}

func treeUtterances(id string) []utteranceData {
	switch id {
	case meetingID1:
		return []utteranceData{
			{speakerIdx: 1, start: 0.0, text: "Good morning, let us get started with the Q1 planning."},
			{speakerIdx: 2, start: 15.5, text: "I have prepared the roadmap items for this quarter."},
			{speakerIdx: 1, start: 40.0, text: "Great, let us go through each milestone in order."},
			{speakerIdx: 2, start: 65.0, text: "The first milestone is the launch of the new dashboard."},
		}
	case meetingID2:
		return []utteranceData{
			{speakerIdx: 1, start: 0.0, text: "Welcome everyone to the product demo session."},
			{speakerIdx: 2, start: 12.0, text: "Thank you. Let me walk you through the new features."},
			{speakerIdx: 1, start: 35.0, text: "The search functionality looks very intuitive."},
			{speakerIdx: 2, start: 52.0, text: "We redesigned the search to reduce the number of clicks."},
		}
	case meetingID3:
		return []utteranceData{
			{speakerIdx: 1, start: 0.0, text: "Today we are reviewing the architecture proposal."},
			{speakerIdx: 2, start: 18.0, text: "I will cover the data layer decisions first."},
			{speakerIdx: 3, start: 42.0, text: "The caching strategy needs to address cache invalidation."},
			{speakerIdx: 1, start: 68.0, text: "Agreed. Let us document the eviction policy."},
			{speakerIdx: 2, start: 90.0, text: "I will write up the ADR after this call."},
		}
	default:
		return []utteranceData{
			{speakerIdx: 1, start: 0.0, text: "Synthetic transcript for meeting " + id + "."},
			{speakerIdx: 2, start: 20.0, text: "Content is generated deterministically for testing."},
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
