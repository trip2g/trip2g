namespace $ {
	$mol_style_define($trip2g_editor_diff, {
		flex: {
			basis: '400px',
			shrink: 0,
			grow: 0,
		},
		minHeight: 0,
		borderLeft: `1px solid ${$mol_theme.line}`,

		Stats: {
			padding: $mol_gap.space,
			borderBottom: `1px solid ${$mol_theme.line}`,
		},

		Unified: {
			overflow: 'auto',
			flex: {
				grow: 1,
			},
			flexDirection: 'column',
			fontFamily: 'monospace',
			fontSize: '13px',
			whiteSpace: 'pre',
			padding: $mol_gap.space,
		},
	})
}
