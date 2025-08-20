namespace $.$$ {
	const { rem } = $mol_style_unit

	$mol_style_define($trip2g_admin_waitlisttgbotrequest_catalog, {
		Rows: {
			flex: {
				grow: 1,
			},
		},
		Row_chat_id_labeler: {
			flex: {
				basis: rem(8), // Standard width for IDs
			},
		},
		Row_bot_name_labeler: {
			flex: {
				basis: rem(8), // Standard width for bot names
			},
		},
		Row_created_at_labeler: {
			flex: {
				basis: rem(8), // Standard width for dates
			},
		},
		Row_note_path_labeler: {
			flex: {
				basis: rem(15), // Wider for path content
			},
		},
	})
}