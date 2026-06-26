namespace $ {
	$mol_style_define($trip2g_editor_pane, {
		flex: {
			direction: 'row',
			grow: 1,
		},
		width: '100%',
		height: '100%',
		overflow: 'hidden',

		Navigation: {
			flex: {
				basis: '260px',
				shrink: 0,
				grow: 0,
			},
			minHeight: 0,
			borderRight: `1px solid ${$mol_theme.line}`,
		},

		Editor: {
			flex: {
				basis: 0,
				grow: 1,
			},
			minWidth: 0,
			minHeight: 0,
		},

		UpdateBanner: {
			flex: { shrink: 0 },
			alignItems: 'center',
			gap: $mol_style_unit.rem(0.5),
			padding: $mol_gap.space,
			background: { color: $mol_theme.focus },
			borderBottom: `1px solid ${$mol_theme.line}`,
		},
	})
}
