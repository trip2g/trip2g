package obsidiancanvas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// demoCanvas is the actual docs/demo/telegramnavigation/demo.canvas content.
const demoCanvas = `{
	"nodes":[
		{"id":"intro","type":"text","text":"Welcome!\n\nI'm @rnbwkpr's assistant. What brings you here today?","x":-600,"y":-40,"width":360,"height":140},
		{"id":"pricing","type":"file","file":"demo/telegramnavigation/pricing.md","x":560,"y":-520,"width":320,"height":220},
		{"id":"services","type":"file","file":"demo/telegramnavigation/services.md","x":-240,"y":-760,"width":320,"height":320},
		{"id":"contact","type":"file","file":"demo/telegramnavigation/contact.md","x":400,"y":100,"width":320,"height":220},
		{"id":"plans","type":"file","file":"demo/telegramnavigation/plans.md","x":1240,"y":-710,"width":320,"height":220},
		{"id":"tip","type":"text","text":"Tip: send /start anytime to return here.","x":1240,"y":-920,"width":320,"height":80},
		{"id":"start","type":"text","text":"START","x":-880,"y":-10,"width":160,"height":80,"color":"4"},
		{"id":"site","type":"link","url":"https://trip2g.com","x":1100,"y":50,"width":600,"height":320}
	],
	"edges":[
		{"id":"intro-pricing","fromNode":"intro","fromSide":"right","toNode":"pricing","toSide":"left","label":"See pricing"},
		{"id":"intro-services","fromNode":"intro","fromSide":"right","toNode":"services","toSide":"bottom","label":"What you offer"},
		{"id":"intro-contact","fromNode":"intro","fromSide":"right","toNode":"contact","toSide":"left","label":"Contact"},
		{"id":"pricing-plans","fromNode":"pricing","fromSide":"right","toNode":"plans","toSide":"left","label":"Full plan list"},
		{"id":"pricing-contact","fromNode":"pricing","fromSide":"bottom","toNode":"contact","toSide":"top","label":"Ask a question"},
		{"id":"services-pricing","fromNode":"services","fromSide":"right","toNode":"pricing","toSide":"left","label":"Pricing"},
		{"id":"contact-intro","fromNode":"contact","fromSide":"bottom","toNode":"intro","toSide":"bottom","label":"Start over"},
		{"id":"contact-site","fromNode":"contact","fromSide":"right","toNode":"site","toSide":"left","label":"Open website"},
		{"id":"plans-pricing","fromNode":"plans","fromSide":"bottom","toNode":"pricing","toSide":"right","label":"Back to pricing"},
		{"id":"plans-tip","fromNode":"plans","fromSide":"top","toNode":"tip","toSide":"bottom","label":"Tip"},
		{"id":"start-intro","fromNode":"start","fromSide":"right","toNode":"intro","toSide":"left"}
	]
}`

func TestParse(t *testing.T) {
	c, err := Parse([]byte(demoCanvas))
	require.NoError(t, err)
	require.Len(t, c.Nodes, 8)
	require.Len(t, c.Edges, 11)

	n, ok := c.Node("intro")
	require.True(t, ok)
	require.Equal(t, "text", n.Type)

	n, ok = c.Node("pricing")
	require.True(t, ok)
	require.Equal(t, "file", n.Type)
	require.Equal(t, "demo/telegramnavigation/pricing.md", n.File)
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := Parse([]byte(`{invalid`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse canvas")
}

func TestEntry_SingleEdge(t *testing.T) {
	// START node has exactly one outgoing edge -> returns target
	c, err := Parse([]byte(demoCanvas))
	require.NoError(t, err)
	require.Equal(t, "intro", c.Entry())
}

func TestEntry_MultiEdge(t *testing.T) {
	// START with multiple outgoing edges -> returns START itself
	raw := `{
		"nodes":[
			{"id":"s","type":"text","text":"START"},
			{"id":"a","type":"text","text":"A"},
			{"id":"b","type":"text","text":"B"}
		],
		"edges":[
			{"id":"e1","fromNode":"s","toNode":"a"},
			{"id":"e2","fromNode":"s","toNode":"b"}
		]
	}`
	c, err := Parse([]byte(raw))
	require.NoError(t, err)
	require.Equal(t, "s", c.Entry())
}

func TestEntry_Missing(t *testing.T) {
	raw := `{"nodes":[{"id":"a","type":"text","text":"hello"}],"edges":[]}`
	c, err := Parse([]byte(raw))
	require.NoError(t, err)
	require.Empty(t, c.Entry())
}

func TestEntry_CaseInsensitive(t *testing.T) {
	raw := `{
		"nodes":[
			{"id":"s","type":"text","text":"  start  "},
			{"id":"a","type":"text","text":"A"}
		],
		"edges":[{"id":"e1","fromNode":"s","toNode":"a"}]
	}`
	c, err := Parse([]byte(raw))
	require.NoError(t, err)
	require.Equal(t, "a", c.Entry())
}

func TestEdgesFrom(t *testing.T) {
	c, err := Parse([]byte(demoCanvas))
	require.NoError(t, err)

	edges := c.EdgesFrom("intro")
	require.Len(t, edges, 3)
	// Declaration order preserved
	require.Equal(t, "pricing", edges[0].ToNode)
	require.Equal(t, "services", edges[1].ToNode)
	require.Equal(t, "contact", edges[2].ToNode)
}

func TestEdgesFrom_NoEdges(t *testing.T) {
	c, err := Parse([]byte(demoCanvas))
	require.NoError(t, err)
	require.Nil(t, c.EdgesFrom("tip"))
}

func TestEdgeLabel_ExplicitLabel(t *testing.T) {
	c, err := Parse([]byte(demoCanvas))
	require.NoError(t, err)

	edges := c.EdgesFrom("intro")
	require.Equal(t, "See pricing", c.EdgeLabel(edges[0]))
}

func TestEdgeLabel_FallbackToNodeLabel(t *testing.T) {
	c, err := Parse([]byte(demoCanvas))
	require.NoError(t, err)

	// start-intro edge has no label
	edges := c.EdgesFrom("start")
	require.Len(t, edges, 1)
	// Falls back to target node's default label (first line of text)
	label := c.EdgeLabel(edges[0])
	require.Equal(t, "Welcome!", label)
}

func TestDefaultLabel_FileNode(t *testing.T) {
	n := Node{Type: "file", File: "some/path/My Document.md"}
	require.Equal(t, "My Document", DefaultLabel(n))
}

func TestDefaultLabel_TextNode(t *testing.T) {
	n := Node{Type: "text", Text: "First line\nSecond line"}
	require.Equal(t, "First line", DefaultLabel(n))
}

func TestDefaultLabel_TextNode_Long(t *testing.T) {
	n := Node{Type: "text", Text: "This is a very long first line that exceeds forty characters easily"}
	require.Equal(t, "This is a very long first line that exce...", DefaultLabel(n))
}

func TestDefaultLabel_TextNode_Empty(t *testing.T) {
	n := Node{Type: "text", Text: ""}
	require.Equal(t, "(text)", DefaultLabel(n))
}

func TestDefaultLabel_LinkNode(t *testing.T) {
	n := Node{Type: "link", URL: "https://example.com"}
	require.Equal(t, "https://example.com", DefaultLabel(n))
}

func TestDefaultLabel_UnknownType(t *testing.T) {
	n := Node{Type: "group", ID: "g1"}
	require.Equal(t, "g1", DefaultLabel(n))
}
