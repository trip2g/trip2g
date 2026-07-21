namespace $.$$ {
	export class $trip2g_admin_onboardingvault extends $.$trip2g_admin_onboardingvault {
		// A plain link, not a fetch: the browser then sends the session cookie and
		// honours Content-Disposition on its own, with no blob juggling.
		@$mol_mem
		override download_uri(): string {
			const params: string[] = []

			const name = this.vault_name().trim()
			if (name) params.push('name=' + encodeURIComponent(name))

			// Bare presence enables it server-side; an explicit value is not needed.
			if (this.admin_tools()) params.push('enable_admin_graphql')

			const query = params.join('&')
			return '/_system/onboarding-vault' + (query ? '?' + query : '')
		}
	}
}
