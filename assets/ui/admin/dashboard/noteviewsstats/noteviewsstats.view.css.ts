namespace $ {
	const {rem} = $mol_style_unit

	$mol_style_define($trip2g_admin_dashboard_noteviewsstats_name, {
		flex: {
			grow: 1,
			shrink: 1,
			basis: rem(12),
		},
		minWidth: 0,
		wordBreak: 'break-word',
	})

	$mol_style_define($trip2g_admin_dashboard_noteviewsstats_num, {
		flex: {
			grow: 0,
			shrink: 0,
			basis: rem(5),
		},
		justifyContent: 'flex-end',
		textAlign: 'right',
		fontVariantNumeric: 'tabular-nums',
	})
}

namespace $ {
	const {px, rem} = $mol_style_unit

	$mol_style_define($trip2g_admin_dashboard_noteviewsstats, {
		Notes_labeler: {
			margin: {
				top: rem(3.5),
			},
		},
		Readers_labeler: {
			margin: {
				top: rem(3.5),
			},
		},
		Sources_labeler: {
			margin: {
				top: rem(3.5),
			},
		},
		Notes_head: {
			color: $mol_theme.shade,
			boxShadow: 'inset 0 -1px 0 0 var(--mol_theme_line)',
		},
		Readers_head: {
			color: $mol_theme.shade,
			boxShadow: 'inset 0 -1px 0 0 var(--mol_theme_line)',
		},
		Sources_head: {
			color: $mol_theme.shade,
			boxShadow: 'inset 0 -1px 0 0 var(--mol_theme_line)',
		},
		Chart: {
			margin: {
				left: px(5),
			},
		},
		Note_row: {
			alignItems: 'center',
		},
		Reader_row: {
			alignItems: 'center',
		},
		Source_row: {
			alignItems: 'center',
		},
		Note_link: {
			minHeight: 0,
			padding: {
				top: 0,
				right: 0,
				bottom: 0,
				left: 0,
			},
		},
		Reader_link: {
			minHeight: 0,
			padding: {
				top: 0,
				right: 0,
				bottom: 0,
				left: 0,
			},
		},
	})
}
