namespace $ {
	const { rem } = $mol_style_unit

	// A chain is a handful of deliveries on one wall clock: each bar spans from
	// the moment the delivery row appeared to the moment its run ended, and the
	// solid part inside it is the run. What is left over — the pale head of the
	// bar — is time spent waiting in the queue, which is often most of a chain.
	$mol_style_define( $trip2g_admin_deliverytrace_timeline, {
		position: 'relative',
		display: 'block',
		flex: { shrink: 0 },
		margin: { top: rem( 0.5 ), bottom: rem( 0.5 ) },
		padding: $mol_gap.text,
		minHeight: rem( 5 ),

		Bar: {
			position: 'absolute',
			display: 'flex',
			alignItems: 'center',
			height: rem( 1.5 ),
			minWidth: rem( 0.5 ),
			padding: '0',
			background: { color: $mol_theme.line },
			border: { radius: rem( 0.25 ) },
			overflow: 'hidden',

			Bar_run: {
				position: 'absolute',
				top: '0',
				bottom: '0',
				right: '0',
				background: { color: $mol_theme.focus },
				border: { radius: rem( 0.25 ) },
			},

			Bar_label: {
				position: 'relative',
				padding: { left: rem( 0.25 ), right: rem( 0.25 ) },
				color: $mol_theme.card,
				font: { size: rem( 0.75 ) },
				whiteSpace: 'nowrap',
				pointerEvents: 'none',
			},
		},
	} )
}
