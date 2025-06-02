namespace $.$$ {
	$mol_style_define( $trip2g_admin_purchase_catalog, {
		
		Purchase_view: {
			display: 'grid',
			gridTemplateColumns: 'minmax(100px, 1fr) minmax(150px, 1fr) minmax(200px, 2fr) minmax(100px, 1fr) minmax(100px, 1fr) minmax(80px, 1fr)',
			gap: '1rem',
			padding: '1rem',
			alignItems: 'center',
			borderBottom: '1px solid var(--mol_theme_line)',
		},

		Purchase_id: {
			fontFamily: 'monospace',
			fontSize: '0.875rem',
		},

		Purchase_email: {
			color: 'var(--mol_theme_text)',
		},

		Purchase_status: {
			padding: '0.25rem 0.5rem',
			borderRadius: '0.25rem',
			fontSize: '0.875rem',
			fontWeight: 'bold',
			textAlign: 'center',
		},

		Purchase_offer_id: {
			fontFamily: 'monospace',
			fontSize: '0.875rem',
		},

	})
}