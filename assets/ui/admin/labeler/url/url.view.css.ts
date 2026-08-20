namespace $ {
	const { rem } = $mol_style_unit

	// Delivery URLs are long — a fleet callback embeds a sha256 segment and a
	// base64 role key — so the cell clips instead of stretching the row past the
	// viewport. minWidth:0 is what lets a flex item shrink below its content.
	$mol_style_define( $trip2g_admin_labeler_url, {
		flex: {
			basis: rem( 20 ),
		},
		minWidth: 0,
		Cell: {
			maxWidth: '100%',
			overflow: 'hidden',
			whiteSpace: 'nowrap',
			textOverflow: 'ellipsis',
		},
	} )
}
