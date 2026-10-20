package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHighlightHTML(t *testing.T) {
	segments := []HighlightSegment{
		{Text: "…a <b> & \"q\" "},
		{Text: "капибара's", Match: true},
		{Text: " end…"},
	}

	require.Equal(t, "…a &lt;b&gt; &amp; &#34;q&#34; <mark>капибара&#39;s</mark> end…", HighlightHTML(segments))
	require.Equal(t, "…a <b> & \"q\" капибара's end…", SegmentsText(segments))
	require.Empty(t, HighlightHTML(nil))
}

func TestPlainSegments(t *testing.T) {
	require.Nil(t, PlainSegments(""))
	require.Equal(t, []HighlightSegment{{Text: "x"}}, PlainSegments("x"))
}
