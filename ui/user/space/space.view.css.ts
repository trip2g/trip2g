namespace $.$$ {
	$mol_style_define( $trip2g_user_space, {
		alignItems: 'center',
		flexWrap: 'wrap',

		Dialog: {
			position: 'relative',
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
