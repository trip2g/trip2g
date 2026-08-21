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
	// Frontmatter is the role note's OWN frontmatter, as flat strings — trip2g
	// stringifies note meta, so a numeric or boolean field arrives as its text
	// form and nothing downstream should expect typed values. It lets a role read
	// its configuration from the note that declares it instead of hardcoding it
	// in the body.
	Frontmatter map[string]string `json:"frontmatter"`

	// Unseal names the frontmatter fields whose values are sealed, and
	// UnsealEnvKey names the env var holding the key that opens them (empty =
	// the default). Both are declared by the role and set by fleet, which holds
	// no key and never sees a plaintext: codellm resolves the key from its OWN
	// environment and decrypts.
	//
	// Declaring the fields explicitly, rather than sniffing values for a sealed
	// prefix, is what makes a typo in a blob fail loudly at unseal time instead
	// of travelling on as an ordinary string and returning a 401 from the
	// upstream API minutes later.
	Unseal       []string `json:"unseal,omitempty"`
	UnsealEnvKey string   `json:"unseal_env_key,omitempty"`

	// Secrets carries the opened values, keyed by frontmatter field name. It is
	// filled by codellm just before the bag is handed to the code child; fleet
	// always leaves it empty. It is deliberately NOT merged into Frontmatter:
	// the bag is what a role dumps while debugging, and keeping secrets out of
	// the object most likely to be printed whole narrows that accident.
	Secrets map[string]string `json:"secrets,omitempty"`

	// EnvPassthrough / EnvPrefix are the role's declaration of which env vars
	// its code needs. codellm intersects them with its OWN operator allowlist,
	// so a role can only narrow what it receives, never widen it. Declaring
	// neither means the whole allowlist, which is what roles written before the
	// fields existed expect.
	EnvPassthrough []string `json:"env_passthrough,omitempty"`
	EnvPrefix      []string `json:"env_prefix,omitempty"`

	ChangedFiles  []ChangeInfo   `json:"changed_files"`
	ChangeFile    *ChangeInfo    `json:"change_file,omitempty"`
	AttachedNotes []AttachedNote `json:"attached_notes"`
	Depth         int            `json:"depth"`
	Now           *string        `json:"now,omitempty"`
}
