package frontmatteredit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// verify is the wall behind the splice: each case is a result the splice could
// have produced if it misjudged a key's lines.
func TestVerifyRefusesABadSplice(t *testing.T) {
	before := "---\ntitle: T\nstatus: draft\ntags: [x]\n---\nBody\n"
	tests := []struct {
		name  string
		after string
		set   map[string]any
		unset []string
	}{
		{
			name:  "an untouched key is lost",
			after: "---\ntitle: T\nstatus: done\n---\nBody\n",
			set:   map[string]any{"status": "done"},
		},
		{
			name:  "the set key appears twice",
			after: "---\ntitle: T\nstatus: done\nstatus: draft\ntags: [x]\n---\nBody\n",
			set:   map[string]any{"status": "done"},
		},
		{
			name:  "the unset key stays",
			after: "---\ntitle: T\nstatus: draft\ntags: [x]\n---\nBody\n",
			unset: []string{"tags"},
		},
		{
			name:  "an untouched value changes",
			after: "---\ntitle: T\nstatus: done\ntags: [y]\n---\nBody\n",
			set:   map[string]any{"status": "done"},
		},
		{
			name:  "the set key gets another value",
			after: "---\ntitle: T\nstatus: draft\ntags: [x]\n---\nBody\n",
			set:   map[string]any{"status": "done"},
		},
		{
			name:  "a key nobody named appears",
			after: "---\ntitle: T\nstatus: done\ntags: [x]\nextra: 1\n---\nBody\n",
			set:   map[string]any{"status": "done"},
		},
		{
			name:  "the body changes",
			after: "---\ntitle: T\nstatus: done\ntags: [x]\n---\nBody\nmore\n",
			set:   map[string]any{"status": "done"},
		},
		{
			name:  "the result is not YAML",
			after: "---\ntitle: T\nstatus: [done\ntags: [x]\n---\nBody\n",
			set:   map[string]any{"status": "done"},
		},
		{
			name:  "the result loses its closing fence",
			after: "---\ntitle: T\nstatus: done\ntags: [x]\nBody\n",
			set:   map[string]any{"status": "done"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verify(before, tt.after, tt.set, tt.unset)
			require.ErrorIs(t, err, ErrUnsafeEdit)
		})
	}
}

func TestVerifyAcceptsTheIntendedEdit(t *testing.T) {
	before := "---\ntitle: T\nstatus: draft\ntags: [x]\n---\nBody\n"
	after := "---\ntitle: T\ntags: [x]\nweight: 2\n---\nBody\n"

	err := verify(before, after, map[string]any{"weight": 2}, []string{"status"})
	require.NoError(t, err)
}

func TestCompareGoldmarkKeysRefusesWhatYAMLv2ReadsElsewhere(t *testing.T) {
	err := compareGoldmarkKeys("~: x\n", map[string]string{"~": "x\n"})
	require.ErrorIs(t, err, ErrUnsafeEdit)
}
