// Package obsidiancanvas provides pure parsing of Obsidian .canvas JSON files
// into a navigable graph structure. No business logic, no Telegram knowledge.
package obsidiancanvas

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Node represents a single node in an Obsidian canvas.
type Node struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // "text" | "file" | "link" | "group"
	Text  string `json:"text,omitempty"`
	File  string `json:"file,omitempty"`
	URL   string `json:"url,omitempty"`
	Color string `json:"color,omitempty"`
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	ID       string `json:"id"`
	FromNode string `json:"fromNode"`
	ToNode   string `json:"toNode"`
	Label    string `json:"label,omitempty"`
}

// Canvas is a parsed Obsidian .canvas file with prebuilt indexes for navigation.
type Canvas struct {
	Nodes []Node
	Edges []Edge

	nodesByID map[string]Node
	outgoing  map[string][]Edge
}

// canvasJSON is the raw JSON structure of a .canvas file.
type canvasJSON struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Parse decodes an Obsidian .canvas JSON and builds navigation indexes.
func Parse(raw []byte) (*Canvas, error) {
	var cf canvasJSON
	if err := json.Unmarshal(raw, &cf); err != nil {
		return nil, fmt.Errorf("parse canvas: %w", err)
	}
	c := &Canvas{
		Nodes:     cf.Nodes,
		Edges:     cf.Edges,
		nodesByID: make(map[string]Node, len(cf.Nodes)),
		outgoing:  make(map[string][]Edge),
	}
	for _, n := range cf.Nodes {
		c.nodesByID[n.ID] = n
	}
	for _, e := range cf.Edges {
		c.outgoing[e.FromNode] = append(c.outgoing[e.FromNode], e)
	}
	return c, nil
}

// Node returns a node by ID.
func (c *Canvas) Node(id string) (Node, bool) {
	n, ok := c.nodesByID[id]
	return n, ok
}

// EdgesFrom returns all edges originating from the given node, in declaration order.
func (c *Canvas) EdgesFrom(nodeID string) []Edge {
	return c.outgoing[nodeID]
}

// Entry finds the START marker node and resolves the entry point.
// Convention: a text node whose trimmed text equals "START" (case-insensitive).
// If START has exactly one outgoing edge, returns that edge's target node ID.
// If START has multiple outgoing edges, returns the START node ID itself (caller renders a menu).
// Returns "" if no START node found.
func (c *Canvas) Entry() string {
	var startID string
	for _, n := range c.nodesByID {
		if n.Type == "text" && strings.EqualFold(strings.TrimSpace(n.Text), "START") {
			startID = n.ID
			break
		}
	}
	if startID == "" {
		return ""
	}
	out := c.outgoing[startID]
	if len(out) == 1 {
		return out[0].ToNode
	}
	return startID
}

// EdgeLabel returns the edge's explicit label, falling back to the target node's
// default label when the edge label is empty.
func (c *Canvas) EdgeLabel(e Edge) string {
	if strings.TrimSpace(e.Label) != "" {
		return e.Label
	}
	if target, ok := c.nodesByID[e.ToNode]; ok {
		return DefaultLabel(target)
	}
	return "-> " + e.ToNode
}

// DefaultLabel produces a human-readable label for a node when no explicit
// edge label is provided.
func DefaultLabel(n Node) string {
	switch n.Type {
	case "file":
		base := filepath.Base(n.File)
		return strings.TrimSuffix(base, filepath.Ext(base))
	case "text":
		for _, line := range strings.Split(n.Text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if len(line) > 40 {
				line = line[:40] + "..."
			}
			return line
		}
		return "(text)"
	case "link":
		return n.URL
	}
	return n.ID
}
