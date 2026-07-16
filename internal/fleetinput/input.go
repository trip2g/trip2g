// Package fleetinput defines the non-secret delivery context shared by Fleet
// and codellm. It is deliberately independent from either transport layer.
package fleetinput

type ChangeInfo struct {
	Path    string `json:"path"`
	Event   string `json:"event"`
	PathID  int64  `json:"path_id"`
	Version int64  `json:"version"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type AttachedNote struct {
	Path      string            `json:"path"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	UpdatedAt string            `json:"updated_at"`
	Tags      []string          `json:"tags"`
	Meta      map[string]string `json:"meta"`
}

type Input struct {
	ChangedFiles  []ChangeInfo   `json:"changed_files"`
	ChangeFile    *ChangeInfo    `json:"change_file,omitempty"`
	AttachedNotes []AttachedNote `json:"attached_notes"`
	Depth         int            `json:"depth"`
	Now           *string        `json:"now,omitempty"`
}
