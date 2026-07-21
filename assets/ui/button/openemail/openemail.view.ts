namespace $.$$ {

	// app is the provider's mobile deep link. The webmail urls below open a
	// browser tab on a phone, which is the wrong place to read a sign-in code,
	// and the #search fragment does not survive the trip at all.
	const providers: Record<string, { url: string; app?: string; search?: (from: string) => string }> = {
		'gmail.com': {
			url: 'https://mail.google.com/mail/u/0/#search/',
			app: 'googlegmail://',
			search: () => `subject:Sign-in Code`,
		},
		'googlemail.com': {
			url: 'https://mail.google.com/mail/u/0/#search/',
			app: 'googlegmail://',
			search: () => `subject:Sign-in Code`,
		},
		'outlook.com': {
			url: 'https://outlook.live.com/mail/0/',
			app: 'ms-outlook://',
		},
		'hotmail.com': {
			url: 'https://outlook.live.com/mail/0/',
			app: 'ms-outlook://',
		},
		'yahoo.com': {
			url: 'https://mail.yahoo.com/',
			app: 'ymail://',
		},
		'icloud.com': {
			url: 'https://www.icloud.com/mail/',
		},
		'me.com': {
			url: 'https://www.icloud.com/mail/',
		},
		'proton.me': {
			url: 'https://mail.proton.me/',
			app: 'protonmail://',
		},
		'protonmail.com': {
			url: 'https://mail.proton.me/',
			app: 'protonmail://',
		},
		'example.com': {
			url: 'https://example.com/mail/',
		},
	}

	const mobile = () => /iPhone|iPad|iPod|Android/i.test(navigator.userAgent ?? '')
	const ios = () => /iPhone|iPad|iPod/i.test(navigator.userAgent ?? '')

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
			if (mobile()) {
				// Fall back to the system mail app on iOS: better a working inbox
				// than a webmail tab the user has to sign into again.
				if (p.app) return p.app
				if (ios()) return 'message://'
			}
			if (p.search) return p.url + encodeURIComponent(p.search(''))
			return p.url
		}

		override sub() {
			if (!this.provider()) return []
			return super.sub()
		}
	}
}
