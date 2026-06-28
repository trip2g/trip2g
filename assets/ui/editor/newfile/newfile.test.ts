namespace $ {
	$mol_test({
		'$trip2g_editor_newfile_normalize: appends .md when missing'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_normalize('notes/idea', []),
				{ ok: true, path: 'notes/idea.md' },
			)
		},
		'$trip2g_editor_newfile_normalize: keeps existing .md extension'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_normalize('readme.md', []),
				{ ok: true, path: 'readme.md' },
			)
		},
		'$trip2g_editor_newfile_normalize: .md match is case-insensitive'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_normalize('NOTE.MD', []),
				{ ok: true, path: 'NOTE.MD' },
			)
		},
		'$trip2g_editor_newfile_normalize: trims whitespace and a leading slash'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_normalize('  /a/b  ', []),
				{ ok: true, path: 'a/b.md' },
			)
		},
		'$trip2g_editor_newfile_normalize: empty input is rejected'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_normalize('   ', []),
				{ ok: false, error: 'empty' },
			)
		},
		'$trip2g_editor_newfile_normalize: existing path is rejected'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_normalize('a/b', ['a/b.md']),
				{ ok: false, error: 'exists' },
			)
		},
		'$trip2g_editor_newfile_normalize: existing check is case-insensitive'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_normalize('Notes/Idea.md', ['notes/idea.md']),
				{ ok: false, error: 'exists' },
			)
		},
		'$trip2g_editor_newfile_initial_content: H1 from basename without extension'() {
			$mol_assert_equal(
				$trip2g_editor_newfile_initial_content('dir/My Note.md'),
				'# My Note\n',
			)
		},
	})
}
