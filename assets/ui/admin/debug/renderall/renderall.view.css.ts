namespace $ {
	const { rem, per } = $mol_style_unit

	$mol_style_define( $trip2g_admin_debug_renderall, {

		Stats: {
			padding: rem( .5 ),
			color: $mol_theme.shade,
		},

		Grid: {
			display: 'flex',
			flex: {
				wrap: 'wrap',
			},
			gap: rem( .5 ),
			padding: rem( .5 ),
		},

		Cell: {
			display: 'flex',
			flex: {
				direction: 'column',
				grow: 1,
				basis: rem( 30 ),
			},
			border: {
				radius: '4px',
			},
			background: {
				color: $mol_theme.card,
			},
		},

		Cell_head: {
			display: 'flex',
			justifyContent: 'space-between',
			alignItems: 'center',
			padding: rem( .25 ),
		},

		Cell_status: {
			padding: $mol_gap.text,
			color: $mol_theme.shade,
		},

		Frame: {
			width: per( 100 ),
			height: rem( 30 ),
			background: {
				color: $mol_theme.back,
			},
		},

	} )
}
