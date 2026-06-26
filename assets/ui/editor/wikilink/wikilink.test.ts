namespace $ {
	$mol_test({
		'$trip2g_editor_wikilink: caret inside a plain link returns its target'() {
			$mol_assert_equal($trip2g_editor_wikilink('Click [[Link]] here', 8), 'Link')
		},

		'$trip2g_editor_wikilink: alias suffix is stripped'() {
			$mol_assert_equal($trip2g_editor_wikilink('see [[Path/Note|Alias]] x', 10), 'Path/Note')
		},

		'$trip2g_editor_wikilink: anchor suffix is stripped'() {
			$mol_assert_equal($trip2g_editor_wikilink('[[Path#Section]]', 5), 'Path')
		},

		'$trip2g_editor_wikilink: anchor and alias together keep only the path'() {
			$mol_assert_equal($trip2g_editor_wikilink('[[Path#Sec|Alias]]', 5), 'Path')
		},

		'$trip2g_editor_wikilink: inner whitespace is trimmed'() {
			$mol_assert_equal($trip2g_editor_wikilink('[[  Link  ]]', 5), 'Link')
		},

		'$trip2g_editor_wikilink: caret outside any link returns null'() {
			$mol_assert_equal($trip2g_editor_wikilink('a [[Link]] b', 0), null)
		},

		'$trip2g_editor_wikilink: with several links, the one under the caret wins'() {
			$mol_assert_equal($trip2g_editor_wikilink('a [[One]] b [[Two]] c', 15), 'Two')
		},
	})
}
