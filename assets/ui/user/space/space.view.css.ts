namespace $.$$ {
	$mol_style_define( $trip2g_user_space, {
		alignItems: 'center',
		flexWrap: 'wrap',

		Dialog: {
			position: 'relative',
		},

		// Let the auth wrapper + book2 catalog shrink to the dialog width instead
		// of overflowing it (book2's natural deck width exceeds the 800px dialog,
		// which clipped the right edge of every tab). min-width:0 is required for
		// flex children to shrink below their content size.
		AuthWrapper: {
			flex: { grow: 1, shrink: 1, basis: 'auto' },
			minWidth: '0',
			maxWidth: '100%',
		},

		CloseButton: {
			position: 'absolute',
			top: $mol_gap.block,
			right: $mol_gap.block,
			cursor: 'pointer',
		},

		Home: {
			flexGrow: '1',
		},

		Email: {
			padding: '0.5rem',
		},

		Content: {
			minWidth: '0',
			maxWidth: '100%',
			Placeholder: {
				flexGrow: '0',
				flexBasis: '0',
			}
		}
	})

	$mol_style_define( $trip2g_user_space_name, {
		margin: '1rem',
	})
}
