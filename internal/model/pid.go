package model

import (
	"encoding/json"
	"strconv"
	"strings"
)

// PID is a note id that tolerates what models actually send: a JSON number,
// a numeric string ("70"), or a non-numeric string like a chunk match_id
// ("p36:c2") or a path. Non-numeric input parses to Value 0 with Raw kept for
// error messages, so a valid path in the same call can still resolve instead
// of the whole request failing on unmarshal.
type PID struct {
	Value int64
	Raw   string
}

func (p *PID) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var raw string
		if err := json.Unmarshal(b, &raw); err != nil {
			return err
		}
		p.Raw = raw
		if v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil {
			p.Value = v
		}
		return nil
	}
	return json.Unmarshal(b, &p.Value)
}

// MarshalJSON emits null for an absent id rather than 0. The generated wire
// encoder cannot evaluate omitempty on a struct field, so it always writes the
// key; null is the one form every peer decodes without error, whether it reads
// note_id as a string, an int64, or a PID.
func (p PID) MarshalJSON() ([]byte, error) {
	if p.Value == 0 {
		// Raw is kept for error messages but is not a usable id, so a
		// non-numeric pid crosses the wire as absent, not as a bogus 0.
		return []byte("null"), nil
	}
	return json.Marshal(p.Value)
}

// IsZero reports whether no id was supplied, so encoding/json's omitzero and
// the handlers agree on what "absent" means.
func (p PID) IsZero() bool {
	return p.Value == 0 && p.Raw == ""
}
