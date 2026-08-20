namespace $ {
	// width:100% plus a shrinkable head is what makes the middle ellipsis work: mol
	// views do not shrink by default, so without this the path simply overflows its
	// cell and gets chopped by the pane instead of losing its middle.
	$mol_style_define( $trip2g_admin_path, {
		display: 'flex',
		width: '100%',
		minWidth: 0,
		maxWidth: '100%',
		flex: { shrink: 1 },

		Head: {
			overflow: 'hidden',
			textOverflow: 'ellipsis',
			whiteSpace: 'nowrap',
			minWidth: 0,
			flex: { shrink: 1 },
		},

		Tail: {
			whiteSpace: 'nowrap',
			flex: { shrink: 0 },
		},
	} )
}
