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
	const {px} = $mol_style_unit

	$mol_style_define($trip2g_admin_dashboard_noteviewsstats, {
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
