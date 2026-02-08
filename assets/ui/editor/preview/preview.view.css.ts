namespace $ {
	$mol_style_define($trip2g_editor_preview, {
		flex: {
			grow: 1,
			basis: 0,
			direction: 'column',
		},
		border: {
			left: {
				width: '1px',
				style: 'solid',
				color: $mol_theme.line,
			},
		},
		overflow: 'auto',
		display: 'flex',

		Head: {
			padding: $mol_gap.block,
			font: {
				weight: 'bold',
			},
			flex: {
				shrink: 0,
			},
		},

		Body: {
			flex: {
				grow: 1,
			},
			padding: $mol_gap.block,
			overflow: 'auto',
		},
	})
}
