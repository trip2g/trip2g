namespace $.$$ {
	const { rem } = $mol_style_unit

	$mol_style_define($trip2g_user_space_subscriptions, {
		List: {
			flex: {
				grow: 1,
			},
		},
		Page: {
			flex: {
				grow: 1,
			},
		},
		Row: {
			border: {
				bottom: {
					style: 'solid',
					width: rem(0.1),
					color: $mol_theme.line,
				}
			}
		},
		Name_labeler: {
			flex: {
				grow: 1,
			},
		},
		CreatedAt_labeler: {
			flex: {
				basis: rem(6),
			},
		},
		ExpiresAt_labeler: {
			flex: {
				basis: rem(6),
			},
		},
	})
}