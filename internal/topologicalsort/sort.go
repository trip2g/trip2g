// Package topologicalsort implements reverse topological sorting for posts.
//
//nolint:cyclop // Topological sort algorithm is inherently complex
package topologicalsort

import (
	"trip2g/internal/model"
)

// ReverseSort performs reverse topological sort on posts to minimize the number
// of updates needed after publishing. Posts that are referenced by others
// are sorted to be published first.
//
// For cycles, the algorithm breaks them optimally to minimize updates.
//
//nolint:gocognit // Topological sort algorithm is inherently complex
func ReverseSort(nvs *model.NoteViews, ids []int64) []int64 {
	if len(ids) == 0 {
		return ids
	}

	// Build set of ids for quick lookup
	idSet := make(map[int64]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	// Find NoteViews that match the ids
	posts := make(map[int64]*model.NoteView)
	for _, id := range ids {
		nv := nvs.GetByPathID(id)
		if nv != nil {
			posts[id] = nv
		}
	}

	// Build graph: outgoing[A] = list of posts A references
	// Build reverse graph: incoming[B] = list of posts that reference B
	outgoing := make(map[int64][]int64)
	incoming := make(map[int64][]int64)

	// Initialize maps
	for id := range posts {
		outgoing[id] = []int64{}
		incoming[id] = []int64{}
	}

	// Build graph using InLinks (posts that reference this post)
	for id, nv := range posts {
		// For each post that references this note
		for referrerPermalink := range nv.InLinks {
			// Resolve permalink to note
			referrerNV, ok := nvs.Map[referrerPermalink]
			if !ok {
				continue
			}

			// Check if referrer is in our batch
			if !idSet[referrerNV.PathID] {
				continue
			}

			// Add edge: referrer -> current note
			// (referrer references this note, so this note should be published first)
			outgoing[referrerNV.PathID] = append(outgoing[referrerNV.PathID], id)
			incoming[id] = append(incoming[id], referrerNV.PathID)
		}
	}

	// Calculate out_degree for each post
	outDegree := make(map[int64]int)
	for id := range posts {
		outDegree[id] = len(outgoing[id])
	}

	// Kahn's algorithm (modified for out_degree instead of in_degree)
	// Start with posts that have out_degree = 0
	var queue []int64
	for id := range posts {
		if outDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	var result []int64
	processed := make(map[int64]bool)

	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]

		result = append(result, current)
		processed[current] = true

		// For all posts that reference current post
		for _, referrer := range incoming[current] {
			if processed[referrer] {
				continue
			}

			// Decrease out_degree
			outDegree[referrer]--
			if outDegree[referrer] == 0 {
				queue = append(queue, referrer)
			}
		}
	}

	// Handle cycles: if there are unprocessed posts
	for len(processed) < len(posts) {
		// Find post with minimum out_degree among unprocessed
		// If multiple posts have same out_degree, pick the one with smallest PathID (deterministic)
		minDegree := -1
		var minID int64

		for id := range posts {
			if processed[id] {
				continue
			}

			if minDegree == -1 || outDegree[id] < minDegree || (outDegree[id] == minDegree && id < minID) {
				minDegree = outDegree[id]
				minID = id
			}
		}

		// Add to result and process
		result = append(result, minID)
		processed[minID] = true

		// Update out_degree for posts referencing this one
		for _, referrer := range incoming[minID] {
			if !processed[referrer] {
				outDegree[referrer]--
			}
		}
	}

	return result
}
