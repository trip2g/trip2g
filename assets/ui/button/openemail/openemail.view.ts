namespace $.$$ {

	const providers: Record<string, { url: string; search?: (from: string) => string }> = {
		'gmail.com': {
			url: 'https://mail.google.com/mail/u/0/#search/',
			search: () => `subject:Sign-in Code`,
		},
		'googlemail.com': {
			url: 'https://mail.google.com/mail/u/0/#search/',
			search: () => `subject:Sign-in Code`,
		},
		'outlook.com': {
			url: 'https://outlook.live.com/mail/0/',
		},
		'hotmail.com': {
			url: 'https://outlook.live.com/mail/0/',
		},
		'yahoo.com': {
			url: 'https://mail.yahoo.com/',
		},
		'icloud.com': {
			url: 'https://www.icloud.com/mail/',
		},
		'me.com': {
			url: 'https://www.icloud.com/mail/',
		},
		'proton.me': {
			url: 'https://mail.proton.me/',
		},
		'protonmail.com': {
			url: 'https://mail.proton.me/',
		},
		'example.com': {
			url: 'https://example.com/mail/',
		},
	}

	export class $trip2g_button_openemail extends $.$trip2g_button_openemail {

		provider() {
			const email = this.email()
			if (!email) return null
			const domain = email.split('@')[1]?.toLowerCase()
			if (!domain) return null
			return providers[domain] ?? null
		}

		override email_url() {
			const p = this.provider()
			if (!p) return ''
			if (p.search) return p.url + encodeURIComponent(p.search(''))
			return p.url
		}

		override sub() {
			if (!this.provider()) return []
			return super.sub()
		}
	}
}
