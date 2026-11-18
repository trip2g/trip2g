namespace $ {
	const make = ( html: string ) => {
		const root = $mol_dom_context.document.createElement( 'div' )
		root.innerHTML = html
		return root.querySelector( '[data-target]' )
	}

	$mol_test({
		'$trip2g_form_enter: a single-line text input submits'() {
			$mol_assert_equal( $trip2g_form_enter( make( '<input data-target>' ) ), true )
		},

		'$trip2g_form_enter: email and password inputs submit'() {
			$mol_assert_equal( $trip2g_form_enter( make( '<input type="email" data-target>' ) ), true )
			$mol_assert_equal( $trip2g_form_enter( make( '<input type="password" data-target>' ) ), true )
		},

		'$trip2g_form_enter: a textarea keeps Enter for a new line'() {
			$mol_assert_equal( $trip2g_form_enter( make( '<textarea data-target></textarea>' ) ), false )
		},

		'$trip2g_form_enter: a checkbox and a button do not submit'() {
			$mol_assert_equal( $trip2g_form_enter( make( '<input type="checkbox" data-target>' ) ), false )
			$mol_assert_equal( $trip2g_form_enter( make( '<button data-target></button>' ) ), false )
		},

		'$trip2g_form_enter: the filter input of an open picker does not submit'() {
			$mol_assert_equal( $trip2g_form_enter( make( '<div mol_pop_bubble><input data-target></div>' ) ), false )
		},

		'$trip2g_form_enter: no target does not submit'() {
			$mol_assert_equal( $trip2g_form_enter( null ), false )
		},
	})
}
