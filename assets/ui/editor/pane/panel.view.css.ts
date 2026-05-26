namespace $ {
	$mol_style_define($trip2g_editor_pane, {
		flex: {
			direction: 'row',
			grow: 1,
		},
		width: '100%',
		height: '100%',
		overflow: 'hidden',

		Navigator: {
			flex: {
				basis: '260px',
				shrink: 0,
				grow: 0,
			},
			minHeight: 0,
			overflow: 'auto',
			borderRight: `1px solid ${$mol_theme.line}`,
		},

		Content: {
			flex: {
				basis: '0',
				grow: 1,
			},
			minWidth: 0,
			minHeight: 0,
		},

		Preview: {
			flex: {
				basis: '0',
				grow: 1,
			},
			minWidth: 0,
			minHeight: 0,
			overflow: 'auto',
			borderLeft: `1px solid ${$mol_theme.line}`,
		},
	})
}
