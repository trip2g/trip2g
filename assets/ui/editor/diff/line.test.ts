namespace $ {
	$mol_test({
		'$trip2g_editor_diff_line_class: an added line is diff-add'() {
			$mol_assert_equal($trip2g_editor_diff_line_class('+added'), 'diff-add')
		},

		'$trip2g_editor_diff_line_class: a removed line is diff-del'() {
			$mol_assert_equal($trip2g_editor_diff_line_class('-removed'), 'diff-del')
		},

		'$trip2g_editor_diff_line_class: the +++ file header is not an add'() {
			$mol_assert_equal($trip2g_editor_diff_line_class('+++ latest'), '')
		},

		'$trip2g_editor_diff_line_class: the --- file header is not a del'() {
			$mol_assert_equal($trip2g_editor_diff_line_class('--- released'), '')
		},

		'$trip2g_editor_diff_line_class: a context line is unstyled'() {
			$mol_assert_equal($trip2g_editor_diff_line_class(' context'), '')
		},

		'$trip2g_editor_diff_line_class: a hunk header is unstyled'() {
			$mol_assert_equal($trip2g_editor_diff_line_class('@@ -1 +1 @@'), '')
		},

		'$trip2g_editor_diff_line_class: an empty line is unstyled'() {
			$mol_assert_equal($trip2g_editor_diff_line_class(''), '')
		},
	})
}
