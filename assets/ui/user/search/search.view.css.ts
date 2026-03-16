namespace $.$$ {
	$mol_style_define( $trip2g_user_search, {
		flexDirection: 'column',
	})

	$mol_style_define( $trip2g_user_search_panel, {
		flexDirection: 'column',
		backgroundColor: 'transparent',
		marginBottom: '1em',

		ResultCount: {
			marginTop: '1em',
			marginBottom: '1em',
		},

		ResultItem: {
			flexDirection: 'column',
			marginBottom: '2em',
			textDecoration: 'none',
		},

		ItemHeader: {
			// space between
			justifyContent: 'space-between',
			marginBottom: '0.5em',
		},

		Content: {
			whiteSpace: 'pre-line',
		},

		Title: {
			fontSize: '1.5em',
		},
	})
}
