namespace $ {
	$mol_style_define($trip2g_editor, {
		Toolbar: {
			display: 'flex',
			alignItems: 'center',
			gap: $mol_gap.block,
			padding: $mol_gap.block,
			borderBottom: `1px solid ${$mol_theme.line}`,
			flex: {
				shrink: 0,
			},
		},

		CloseButton: {
			marginLeft: 'auto',
		},

		Pane: {
			flex: {
				grow: 1,
			},
			overflow: 'hidden',
		},
	})
}
