namespace $ {
	$mol_test({
		'$trip2g_editor_merge: identical content has no markers'() {
			$mol_assert_equal(
				$trip2g_editor_merge('a\nb\nc', 'a\nb\nc'),
				'a\nb\nc',
			)
		},

		'$trip2g_editor_merge: disjoint changes get separate conflict blocks'() {
			$mol_assert_equal(
				$trip2g_editor_merge('a\nX\nb\nc', 'a\nb\nY\nc'),
				'a\n<<<<<<< mine\nX\n=======\n>>>>>>> updated elsewhere\nb\n<<<<<<< mine\n=======\nY\n>>>>>>> updated elsewhere\nc',
			)
		},

		'$trip2g_editor_merge: a line added on one side keeps common lines unmarked'() {
			$mol_assert_equal(
				$trip2g_editor_merge('a\nb', 'a\nZ\nb'),
				'a\n<<<<<<< mine\n=======\nZ\n>>>>>>> updated elsewhere\nb',
			)
		},

		'$trip2g_editor_merge: common prefix and suffix are not marked'() {
			$mol_assert_equal(
				$trip2g_editor_merge('a\nb\nc\nd', 'a\nXX\nc\nd'),
				'a\n<<<<<<< mine\nb\n=======\nXX\n>>>>>>> updated elsewhere\nc\nd',
			)
		},

		'$trip2g_editor_merge: an empty other side still marks the difference'() {
			$mol_assert_equal(
				$trip2g_editor_merge('a\nb', ''),
				'<<<<<<< mine\na\nb\n=======\n\n>>>>>>> updated elsewhere',
			)
		},
	})
}
