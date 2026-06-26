namespace $ {
	$mol_test({
		'$trip2g_editor_versions_title: assembles version, date and byte size'() {
			$mol_assert_equal(
				$trip2g_editor_versions_title({ version: 3, contentLength: 128 }, '99', '2024-01-02 10:00'),
				'v3 · 2024-01-02 10:00 · 128b',
			)
		},
		'$trip2g_editor_versions_title: not found returns the raw id'() {
			$mol_assert_equal($trip2g_editor_versions_title(undefined, '99', 'ignored'), '99')
		},
		'$trip2g_editor_versions_title: zero version and length still render'() {
			$mol_assert_equal(
				$trip2g_editor_versions_title({ version: 0, contentLength: 0 }, '1', '2024-01-02 10:00'),
				'v0 · 2024-01-02 10:00 · 0b',
			)
		},
		'$trip2g_editor_versions_title: large byte count is shown raw'() {
			$mol_assert_equal(
				$trip2g_editor_versions_title({ version: 12, contentLength: 1048576 }, '12', '2024-06-01 00:00'),
				'v12 · 2024-06-01 00:00 · 1048576b',
			)
		},

		'$trip2g_editor_versions_diff_pair: newest entry diffs against the next/older'() {
			$mol_assert_equal(
				$trip2g_editor_versions_diff_pair([{ versionId: 30 }, { versionId: 20 }, { versionId: 10 }], '30'),
				{ from: 20, to: 30 },
			)
		},
		'$trip2g_editor_versions_diff_pair: middle entry uses the older neighbour'() {
			$mol_assert_equal(
				$trip2g_editor_versions_diff_pair([{ versionId: 30 }, { versionId: 20 }, { versionId: 10 }], '20'),
				{ from: 10, to: 20 },
			)
		},
		'$trip2g_editor_versions_diff_pair: oldest entry has nothing older'() {
			$mol_assert_equal(
				$trip2g_editor_versions_diff_pair([{ versionId: 30 }, { versionId: 20 }, { versionId: 10 }], '10'),
				null,
			)
		},
		'$trip2g_editor_versions_diff_pair: unknown id returns null'() {
			$mol_assert_equal(
				$trip2g_editor_versions_diff_pair([{ versionId: 30 }, { versionId: 20 }], '99'),
				null,
			)
		},
		'$trip2g_editor_versions_diff_pair: single version has no pair'() {
			$mol_assert_equal($trip2g_editor_versions_diff_pair([{ versionId: 42 }], '42'), null)
		},
		'$trip2g_editor_versions_diff_pair: empty history returns null without throwing'() {
			$mol_assert_equal($trip2g_editor_versions_diff_pair([], '42'), null)
		},
	})
}
