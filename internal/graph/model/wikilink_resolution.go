package model

// WikilinkResolution is the GraphQL result of resolveWikilinks. Declared by
// hand (and pinned in gqlgen.yml) because autobind would otherwise match the
// unrelated trip2g/internal/model.WikilinkResolution config enum by name.
type WikilinkResolution struct {
	Link string  `json:"link"`
	Path *string `json:"path,omitempty"`
	URL  *string `json:"url,omitempty"`
}
