namespace $ {
	$mol_style_define( $trip2g_kanban, {
		flexDirection: 'row',
		alignItems: 'flex-start',
		overflowX: 'auto',
		padding: '1rem',
		gap: '1rem',

		Column: {
			flex: { grow: 0, shrink: 0, basis: '17rem' },
			flexDirection: 'column',
			alignItems: 'stretch',
		},

		Column_body: {
			flexDirection: 'column',
			background: 'var(--pico-card-background-color)',
			borderRadius: '0.5rem',
			padding: '0.75rem',
			gap: '0.5rem',
		},

		Column_title: {
			fontWeight: 'bold',
			marginBottom: '0.5rem',
		},

		Add_card: {
			alignSelf: 'stretch',
		},

		Card_body: {
			flexDirection: 'row',
			alignItems: 'flex-start',
			gap: '0.5rem',
			background: 'var(--pico-background-color)',
			borderLeft: '1px solid var(--pico-muted-border-color)',
			borderRight: '1px solid var(--pico-muted-border-color)',
			borderTop: '1px solid var(--pico-muted-border-color)',
			borderBottom: '1px solid var(--pico-muted-border-color)',
			borderRadius: '0.375rem',
			padding: '0.5rem',
			cursor: 'grab',
		},

		Card_text: {
			flex: { grow: 1 },
			background: 'transparent',
		},
	} )
}
