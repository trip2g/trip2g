namespace $ {
	$mol_test({
		'$trip2g_editor_navigator_filter: empty query keeps all in order'() {
			$mol_assert_equal($trip2g_editor_navigator_filter(['a/b.md', 'c/D.md'], ''), ['a/b.md', 'c/D.md'])
		},
		'$trip2g_editor_navigator_filter: whitespace-only query is trimmed to all'() {
			$mol_assert_equal($trip2g_editor_navigator_filter(['a/b.md', 'c/D.md'], '   '), ['a/b.md', 'c/D.md'])
		},
		'$trip2g_editor_navigator_filter: lowercase query matches uppercase path'() {
			$mol_assert_equal($trip2g_editor_navigator_filter(['a/B.md', 'c/d.md'], 'b'), ['a/B.md'])
		},
		'$trip2g_editor_navigator_filter: uppercase query matches lowercase path'() {
			$mol_assert_equal($trip2g_editor_navigator_filter(['readme.md', 'notes/Big.md'], 'BIG'), ['notes/Big.md'])
		},
		'$trip2g_editor_navigator_filter: no match returns empty'() {
			$mol_assert_equal($trip2g_editor_navigator_filter(['a.md', 'b.md'], 'zzz'), [])
		},
		'$trip2g_editor_navigator_filter: query is trimmed then substring-matched'() {
			$mol_assert_equal($trip2g_editor_navigator_filter(['x/Quarterly.md'], ' arter '), ['x/Quarterly.md'])
		},

		'$trip2g_editor_navigator_ids: root prefix keeps everything'() {
			$mol_assert_equal($trip2g_editor_navigator_ids({ 'a/b': ['a/b'], 'c/d': ['c/d'] }, ''), ['a/b', 'c/d'])
		},
		'$trip2g_editor_navigator_ids: only tags under the prefix survive'() {
			$mol_assert_equal($trip2g_editor_navigator_ids({ 'a/b': ['a/b'], 'c/d': ['c/d'] }, 'a'), ['a/b'])
		},
		'$trip2g_editor_navigator_ids: a tag equal to the prefix is excluded'() {
			$mol_assert_equal($trip2g_editor_navigator_ids({ a: ['a'], 'a/b': ['a/b'] }, 'a'), ['a/b'])
		},
		'$trip2g_editor_navigator_ids: prefix matches only at a slash boundary'() {
			$mol_assert_equal($trip2g_editor_navigator_ids({ 'ab/x': ['ab/x'] }, 'a'), [])
		},
		'$trip2g_editor_navigator_ids: nested prefix matches deeper path'() {
			$mol_assert_equal($trip2g_editor_navigator_ids({ 'a/b/c': ['a/b/c'] }, 'a/b'), ['a/b/c'])
		},

		'$trip2g_editor_navigator_subfolders: leaf at this level yields no folder'() {
			$mol_assert_equal(
				$trip2g_editor_navigator_subfolders(
					['a/x.md', 'a/b/y.md'],
					{ 'a/x.md': ['a/x.md'], 'a/b/y.md': ['a/b/y.md'] },
					'a',
				),
				['b'],
			)
		},
		'$trip2g_editor_navigator_subfolders: two files in one subfolder dedupe'() {
			$mol_assert_equal(
				$trip2g_editor_navigator_subfolders(
					['a/b/y.md', 'a/b/z.md'],
					{ 'a/b/y.md': ['a/b/y.md'], 'a/b/z.md': ['a/b/z.md'] },
					'a',
				),
				['b'],
			)
		},
		'$trip2g_editor_navigator_subfolders: root prefix gives sorted top-level folders'() {
			$mol_assert_equal(
				$trip2g_editor_navigator_subfolders(
					['x.md', 'docs/r.md', 'blog/p.md'],
					{ 'x.md': ['x.md'], 'docs/r.md': ['docs/r.md'], 'blog/p.md': ['blog/p.md'] },
					'',
				),
				['blog', 'docs'],
			)
		},
		'$trip2g_editor_navigator_subfolders: all leaves means no folders'() {
			$mol_assert_equal(
				$trip2g_editor_navigator_subfolders(['a/x.md'], { 'a/x.md': ['a/x.md'] }, 'a'),
				[],
			)
		},
		'$trip2g_editor_navigator_subfolders: subfolders come back sorted, not by insertion'() {
			$mol_assert_equal(
				$trip2g_editor_navigator_subfolders(
					['m/c/f', 'm/a/f', 'm/b/f'],
					{ 'm/c/f': ['m/c/f'], 'm/a/f': ['m/a/f'], 'm/b/f': ['m/b/f'] },
					'm',
				),
				['a', 'b', 'c'],
			)
		},
	})
}
