namespace $.$$ {
	$mol_style_define($trip2g_admin_onboardingvault, {
		Body_content: {
			gap: $mol_style_unit.rem(0.75),
		},
		NameField: {
			maxWidth: $mol_style_unit.rem(30),
		},
		AdminToolsNote: {
			color: $mol_theme.shade,
		},
		Download: {
			alignSelf: 'flex-start',
			alignItems: 'center',
			gap: $mol_style_unit.rem(0.5),
			padding: $mol_style_unit.rem(0.5),
			borderRadius: $mol_style_unit.rem(0.25),
			background: { color: $mol_theme.back },
			color: $mol_theme.text,
		},
		Warning: {
			flexDirection: 'column',
			gap: $mol_style_unit.rem(0.25),
			padding: $mol_style_unit.rem(0.75),
			borderRadius: $mol_style_unit.rem(0.5),
			border: { width: '1px', style: 'solid', color: $mol_theme.line },
			borderLeft: `${$mol_style_unit.rem(0.25)} solid ${$mol_theme.focus}`,
		},
		WarningHead: {
			fontWeight: 'bold',
		},
		Automation: {
			flexDirection: 'column',
			gap: $mol_style_unit.rem(0.25),
			color: $mol_theme.shade,
		},
		AutomationHead: {
			fontWeight: 'bold',
		},
		AutomationUri: {
			fontFamily: 'monospace',
			wordBreak: 'break-all',
		},
	})
}
