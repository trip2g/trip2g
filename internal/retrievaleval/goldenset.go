package retrievaleval

import (
	"encoding/json"
	"fmt"
	"os"
)

// GoldenQuery is one labeled query → relevant-note(s) pair.
// Direction is one of: "ru->ru", "en->en", "ru->en", "en->ru".
type GoldenQuery struct {
	Query        string   `json:"query"`
	Lang         string   `json:"lang"`
	Direction    string   `json:"direction"`
	ExpectedURLs []string `json:"expected_urls"`
	Verified     bool     `json:"verified"`
}

type GoldenSet struct {
	Queries []GoldenQuery `json:"queries"`
}

func LoadGoldenSet(path string) (*GoldenSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read golden set: %w", err)
	}
	var gs GoldenSet
	err = json.Unmarshal(data, &gs)
	if err != nil {
		return nil, fmt.Errorf("parse golden set: %w", err)
	}
	err = gs.Validate()
	if err != nil {
		return nil, err
	}
	return &gs, nil
}

// Validate ensures every query has text and at least one expected URL.
func (gs GoldenSet) Validate() error {
	for i, q := range gs.Queries {
		if q.Query == "" {
			return fmt.Errorf("query %d: empty query text", i)
		}
		if len(q.ExpectedURLs) == 0 {
			return fmt.Errorf("query %d (%q): no expected_urls", i, q.Query)
		}
	}
	return nil
}

// Verified returns only hand-verified queries (the trustworthy subset).
func (gs GoldenSet) Verified() []GoldenQuery {
	out := make([]GoldenQuery, 0, len(gs.Queries))
	for _, q := range gs.Queries {
		if q.Verified {
			out = append(out, q)
		}
	}
	return out
}
