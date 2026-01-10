namespace $ {
	$mol_style_define($trip2g_admin_layout_editor, {
		Blocks_section: {
			flex: {
				direction: 'column',
			},
			gap: '0.5rem',
		},
		Blocks_list: {
			border: {
				style: 'dashed',
				width: '1px',
				color: $mol_theme.line,
			},
			padding: '0.5rem',
			minHeight: '3rem',
		},
		Blocks_drop: {
			'@': {
				mol_drop_status: {
					drag: {
						'>': {
							$mol_view: {
								':last-child': {
									boxShadow: `inset 0 -2px 0 0 ${$mol_theme.focus}`,
								},
							},
						},
					},
				},
			},
		},
		Palette: {
			gap: '0.25rem',
		},
		Trash: {
			padding: '1rem',
			border: {
				style: 'dashed',
				width: '1px',
				color: $mol_theme.line,
			},
			justify: {
				content: 'center',
			},
			align: {
				items: 'center',
			},
			gap: '0.5rem',
			color: $mol_theme.shade,
		},
		Trash_drop: {
			'@': {
				mol_drop_status: {
					drag: {
						background: {
							color: $mol_theme.hover,
						},
					},
				},
			},
		},
		Preview_section: {
			flex: {
				direction: 'column',
			},
			gap: '0.5rem',
		},
	})

	$mol_style_define($trip2g_admin_layout_editor_block, {
		Drop: {
			'@': {
				mol_drop_status: {
					drag: {
						boxShadow: `inset 0 2px 0 0 ${$mol_theme.focus}`,
					},
				},
			},
		},
		Content: {
			padding: '0.5rem',
			border: {
				style: 'solid',
				width: '1px',
				color: $mol_theme.line,
			},
			background: {
				color: $mol_theme.card,
			},
			cursor: 'grab',
		},
		Type: {
			font: {
				weight: 'bold',
			},
		},
		Name: {
			color: $mol_theme.shade,
		},
		'@': {
			mol_drag_status: {
				drag: {
					opacity: 0.5,
				},
			},
		},
	})

	$mol_style_define($trip2g_admin_layout_editor_palette_item, {
	})

	$mol_style_define($trip2g_admin_layout_editor_block_form, {
		flex: {
			direction: 'column',
		},
		padding: '0.5rem',
		border: {
			style: 'solid',
			width: '1px',
			color: $mol_theme.focus,
		},
		background: {
			color: $mol_theme.card,
		},
		gap: '0.5rem',
	})
}
