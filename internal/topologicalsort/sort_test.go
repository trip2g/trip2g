package topologicalsort_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/topologicalsort"

	"github.com/stretchr/testify/require"
)

func TestTopologicalSortWithCycle(t *testing.T) {
	// Create 5 notes:
	// Note 1 -> 2 (note1 links to note2)
	// Note 2 -> 3 (note2 links to note3)
	// Note 3 -> 1 (note3 links to note1) - creates cycle
	// Note 4 -> 1 (note4 links to note1)
	// Note 5 - standalone, not in the batch

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path:   "note1.md",
				PathID: 1,
				Content: []byte(`---
free: true
title: "Note 1"
telegram_publish_tags: ["tag"]
---
Link to [[note2]]`),
			},
			{
				Path:   "note2.md",
				PathID: 2,
				Content: []byte(`---
free: true
title: "Note 2"
telegram_publish_tags: ["tag"]
---
Link to [[note3]]`),
			},
			{
				Path:   "note3.md",
				PathID: 3,
				Content: []byte(`---
free: true
title: "Note 3"
telegram_publish_tags: ["tag"]
---
Link to [[note1]]`),
			},
			{
				Path:   "note4.md",
				PathID: 4,
				Content: []byte(`---
free: true
title: "Note 4"
telegram_publish_tags: ["tag"]
---
Link to [[note1]]`),
			},
			{
				Path:   "note5.md",
				PathID: 5,
				Content: []byte(`---
free: true
title: "Note 5"
telegram_publish_tags: ["tag"]
---
Standalone note`),
			},
		},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)
	require.Len(t, nvs.List, 5)

	// Prepare batch for publishing (all except note5)
	// Note: nvs.Map keys are permalinks like "/note1"
	// Input in wrong order: note4 comes before note1
	// Expected: algorithm should put note1 before note4
	ids := []int64{
		nvs.Map["/note4"].PathID,
		nvs.Map["/note3"].PathID,
		nvs.Map["/note2"].PathID,
		nvs.Map["/note1"].PathID,
	}

	// Sort to minimize updates
	sorted := topologicalsort.ReverseSort(nvs, ids)

	// Verify result
	require.Len(t, sorted, 4, "All 4 posts should be in result")

	// Build position map for checking order constraints
	position := make(map[int64]int)
	for i, id := range sorted {
		position[id] = i
	}

	// Verify all posts are present
	require.Contains(t, position, nvs.Map["/note1"].PathID)
	require.Contains(t, position, nvs.Map["/note2"].PathID)
	require.Contains(t, position, nvs.Map["/note3"].PathID)
	require.Contains(t, position, nvs.Map["/note4"].PathID)

	// Key constraint: note4 references note1, so note1 should come before note4
	// This minimizes updates (when we publish note4, note1 is already published)
	require.Less(t, position[nvs.Map["/note1"].PathID], position[nvs.Map["/note4"].PathID],
		"note1 should be published before note4 (note4 references note1)")

	// For the cycle 1->2->3->1:
	// At least one note will need updating, but the order should break the cycle optimally
	// We can't enforce strict order here, but we verify the constraint above holds
}

func TestTopologicalSortLinearChain(t *testing.T) {
	// Create 4 notes in a linear chain:
	// Note 1 -> Note 2 -> Note 3 -> Note 4
	// Expected output: [4, 3, 2, 1] (reverse order)

	mdOptions := mdloader.Options{
		Sources: []mdloader.SourceFile{
			{
				Path:   "note1.md",
				PathID: 1,
				Content: []byte(`---
free: true
title: "Note 1"
telegram_publish_tags: ["tag"]
---
Link to [[note2]]`),
			},
			{
				Path:   "note2.md",
				PathID: 2,
				Content: []byte(`---
free: true
title: "Note 2"
telegram_publish_tags: ["tag"]
---
Link to [[note3]]`),
			},
			{
				Path:   "note3.md",
				PathID: 3,
				Content: []byte(`---
free: true
title: "Note 3"
telegram_publish_tags: ["tag"]
---
Link to [[note4]]`),
			},
			{
				Path:   "note4.md",
				PathID: 4,
				Content: []byte(`---
free: true
title: "Note 4"
telegram_publish_tags: ["tag"]
---
No links`),
			},
		},
		Log:     &logger.TestLogger{},
		Version: "latest",
	}

	nvs, err := mdloader.Load(mdOptions)
	require.NoError(t, err)
	require.Len(t, nvs.List, 4)

	// Input in original order
	ids := []int64{
		nvs.Map["/note1"].PathID,
		nvs.Map["/note2"].PathID,
		nvs.Map["/note3"].PathID,
		nvs.Map["/note4"].PathID,
	}

	// Sort to minimize updates
	sorted := topologicalsort.ReverseSort(nvs, ids)

	// Verify result
	require.Len(t, sorted, 4, "All 4 posts should be in result")

	// Expected order: [4, 3, 2, 1] (reverse of input)
	// Note 4 has no outgoing links, should be first
	// Note 3 links to 4 (already published), should be second
	// Note 2 links to 3 (already published), should be third
	// Note 1 links to 2 (already published), should be last
	require.Equal(t, nvs.Map["/note4"].PathID, sorted[0], "note4 should be first (no outgoing links)")
	require.Equal(t, nvs.Map["/note3"].PathID, sorted[1], "note3 should be second")
	require.Equal(t, nvs.Map["/note2"].PathID, sorted[2], "note2 should be third")
	require.Equal(t, nvs.Map["/note1"].PathID, sorted[3], "note1 should be last")
}
