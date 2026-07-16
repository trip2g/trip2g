namespace $.$$ {
	const { rem } = $mol_style_unit

	$mol_style_define( $trip2g_codellm_debugger_document_prose_block, {
		Control: {
			flexDirection: 'row',
			minWidth: rem( 5 ),
			paddingRight: rem( 1 ),
		}
	} )

	$mol_style_define( $trip2g_codellm_debugger_document_code_block, {
		width: '500px',
		minWidth: '500px',
		maxWidth: '500px',
		flex: { grow: 0, shrink: 0, basis: '500px' },
		Rows: {
			display: 'flex',
			flexDirection: 'column',
			gap: $mol_gap.block,
			minWidth: 0,
		},
		Code: {
			minWidth: 0,
		},
		Stdout: {
			minWidth: 0,
			whiteSpace: 'pre-wrap',
		},
		Stderr: {
			minWidth: 0,
			whiteSpace: 'pre-wrap',
			color: 'red',
		},
		Error: {
			whiteSpace: 'pre-wrap',
			color: 'red',
		},
	} )
}
