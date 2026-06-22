namespace $.$$ {
	$mol_style_define($trip2g_user_space_tokens, {
		TokenList: {
			flexDirection: 'column',
			gap: $mol_style_unit.rem(0.5),
		},
		TokenRow: {
			flexDirection: 'column',
			gap: $mol_style_unit.rem(0.25),
			padding: $mol_style_unit.rem(0.75),
			borderRadius: $mol_style_unit.rem(0.5),
			border: { width: '1px', style: 'solid', color: $mol_theme.line },
		},
		TokenHead: {
			flexDirection: 'row',
			alignItems: 'center',
			justifyContent: 'space-between',
			gap: $mol_style_unit.rem(0.5),
		},
		TokenNameValue: {
			flex: { grow: 1, shrink: 1, basis: 'auto' },
			fontWeight: 'bold',
			overflow: 'hidden',
			textOverflow: 'ellipsis',
			whiteSpace: 'nowrap',
		},
		TokenMeta: {
			flexDirection: 'row',
			flexWrap: 'wrap',
			gap: [ $mol_style_unit.rem(0.25), $mol_style_unit.rem(1.25) ],
		},
		Backdrop: {
			position: 'fixed',
			inset: '0',
			zIndex: 1000,
			display: 'flex',
			justifyContent: 'center',
			alignItems: 'center',
			padding: $mol_style_unit.rem(1),
			background: { color: 'rgba(0, 0, 0, 0.5)' as any },
		},
		PlaintextModal: {
			flexDirection: 'column',
			gap: $mol_style_unit.rem(0.75),
			padding: $mol_style_unit.rem(1),
			borderRadius: $mol_style_unit.rem(0.5),
			maxWidth: $mol_style_unit.rem(38),
			width: '100%',
			maxHeight: '90vh',
			overflow: 'auto',
			background: { color: $mol_theme.back },
			boxShadow: `0 ${$mol_style_unit.rem(0.5)} ${$mol_style_unit.rem(2)} rgba(0, 0, 0, 0.3)`,
		},
		ModalTitle: {
			fontWeight: 'bold',
		},
		PlaintextValue: {
			fontFamily: 'monospace',
			wordBreak: 'break-all',
		},
		CurlExample: {
			fontFamily: 'monospace',
			fontSize: $mol_style_unit.rem(0.85),
			whiteSpace: 'pre-wrap',
			wordBreak: 'break-all',
		},
		McpUrlValue: {
			fontFamily: 'monospace',
			wordBreak: 'break-all',
		},
	})
}
