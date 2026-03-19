package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// sessionFile holds auth credentials and session data persisted between runs.
type sessionFile struct {
	APIID       int    `json:"api_id"`
	APIHash     string `json:"api_hash"`
	Phone       string `json:"phone,omitempty"`
	SessionData string `json:"session_data"` // base64-encoded gotd session bytes
}

func defaultSessionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "session.json"
	}
	return filepath.Join(home, ".config", "exporttgchannel", "session.json")
}

func loadSession(path string) (*sessionFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("session file not found at %s (run 'exporttgchannel auth' first)", path)
	}
	var s sessionFile
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("corrupt session file: %w", err)
	}
	return &s, nil
}

func saveSession(path string, s *sessionFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *sessionFile) sessionBytes() ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s.SessionData)
	if err != nil {
		return nil, fmt.Errorf("invalid session data: %w", err)
	}
	return b, nil
}
