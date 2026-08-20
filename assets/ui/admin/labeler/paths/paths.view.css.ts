namespace $ {
	const { rem } = $mol_style_unit

	$mol_style_define( $trip2g_admin_labeler_paths, {
		flex: {
			basis: rem( 30 ),
		},
		minWidth: 0,

		Cell: {
			padding: $mol_gap.text,
			gap: rem( 0.25 ),
			minWidth: 0,
		},
	} )
}
