namespace $ {
	$mol_style_define($trip2g_editor_newfile, {
		flex: {
			basis: '320px',
			shrink: 0,
			grow: 0,
		},
		minHeight: 0,
		borderLeft: `1px solid ${$mol_theme.line}`,

		Field: {
			padding: $mol_gap.space,
		},

		Hint: {
			padding: $mol_gap.space,
			color: $mol_theme.shade,
			fontSize: '0.85rem',
		},

		Error: {
			padding: $mol_gap.space,
			color: 'tomato',
			fontSize: '0.85rem',
		},
	})
}
