namespace $.$$ {
	const { rem } = $mol_style_unit

	$mol_style_define( $trip2g_codellm_debugger_document, {
		Body_content: {
			maxWidth: rem( 50 ),
			minWidth: rem( 40 ),
			boxSizing: 'border-box',
		},
	} )

	$mol_style_define( $trip2g_codellm_debugger_document_prose_block, {
		Control: {
			flexDirection: 'row',
			minWidth: rem( 5 ),
			paddingRight: rem( 1 ),
		}
	} )

	$mol_style_define( $trip2g_codellm_debugger_document_code_block, {
		Control: {
			flexDirection: 'row',
			alignItems: 'flex-start',
			minWidth: rem( 5 ),
			paddingRight: rem( 1 ),
		},
		PipeControl: {
			minWidth: rem( 5 ),
		},
		Error: {
			color: 'red',
			whiteSpace: 'pre-wrap',
		},
	} )
}
