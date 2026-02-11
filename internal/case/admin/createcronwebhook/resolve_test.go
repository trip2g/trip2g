package createcronwebhook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateBounds(t *testing.T) {
	tests := []struct {
		name           string
		maxDepth       int64
		timeoutSeconds int64
		maxRetries     int64
		wantErr        bool
		wantFieldNames []string
	}{
		{
			name:           "all valid",
			maxDepth:       1,
			timeoutSeconds: 60,
			maxRetries:     3,
			wantErr:        false,
		},
		{
			name:           "maxDepth too low",
			maxDepth:       -1,
			timeoutSeconds: 60,
			maxRetries:     0,
			wantErr:        true,
			wantFieldNames: []string{"maxDepth"},
		},
		{
			name:           "maxDepth too high",
			maxDepth:       1000,
			timeoutSeconds: 60,
			maxRetries:     0,
			wantErr:        true,
			wantFieldNames: []string{"maxDepth"},
		},
		{
			name:           "timeoutSeconds too low",
			maxDepth:       1,
			timeoutSeconds: 0,
			maxRetries:     0,
			wantErr:        true,
			wantFieldNames: []string{"timeoutSeconds"},
		},
		{
			name:           "timeoutSeconds too high",
			maxDepth:       1,
			timeoutSeconds: 3601,
			maxRetries:     0,
			wantErr:        true,
			wantFieldNames: []string{"timeoutSeconds"},
		},
		{
			name:           "maxRetries too low",
			maxDepth:       1,
			timeoutSeconds: 60,
			maxRetries:     -1,
			wantErr:        true,
			wantFieldNames: []string{"maxRetries"},
		},
		{
			name:           "maxRetries too high",
			maxDepth:       1,
			timeoutSeconds: 60,
			maxRetries:     101,
			wantErr:        true,
			wantFieldNames: []string{"maxRetries"},
		},
		{
			name:           "multiple errors",
			maxDepth:       -1,
			timeoutSeconds: 0,
			maxRetries:     -1,
			wantErr:        true,
			wantFieldNames: []string{"maxDepth", "timeoutSeconds", "maxRetries"},
		},
		{
			name:           "boundary value valid: minDepth",
			maxDepth:       0,
			timeoutSeconds: 1,
			maxRetries:     0,
			wantErr:        false,
		},
		{
			name:           "boundary value valid: maxDepth",
			maxDepth:       999,
			timeoutSeconds: 3600,
			maxRetries:     100,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateBounds(tt.maxDepth, tt.timeoutSeconds, tt.maxRetries)

			if tt.wantErr {
				require.NotNil(t, result, "expected error payload")
				require.NotEmpty(t, result.ByFields, "expected field errors")

				gotFieldNames := make([]string, len(result.ByFields))
				for i, fm := range result.ByFields {
					gotFieldNames[i] = fm.Name
				}
				require.ElementsMatch(t, tt.wantFieldNames, gotFieldNames, "field names mismatch")
			} else {
				require.Nil(t, result, "expected no error payload")
			}
		})
	}
}
