namespace $.$$ {
	export class $trip2g_user_space_tokens_show extends $.$trip2g_user_space_tokens_show {
		// Reached as a route, so the id comes from the URL and the screen loads
		// itself instead of being handed a row by the list.
		@$mol_mem
		override token_id(): string {
			return this.$.$mol_state_arg.value('token_id') ?? ''
		}

		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_user_space_tokens_list()
			if (!res.viewer.user) {
				return null
			}

			return res.viewer.user.tokens.find(token => token.id === this.token_id()) ?? null
		}

		override token_name(): string {
			return this.data()?.name ?? '-'
		}

		override token_prefix(): string {
			return this.data()?.tokenPrefix ?? '-'
		}

		override token_created_at() {
			return this.data()?.createdAt ?? null
		}

		override token_last_used_at() {
			return this.data()?.lastUsedAt ?? null
		}

		override token_expires_at() {
			return this.data()?.expiresAt ?? null
		}

		override token_revoked_at() {
			return this.data()?.revokedAt ?? null
		}

		// Revoking rewrites the row this screen reads, so it reloads rather than
		// showing a state the server no longer agrees with.
		override revoked(next?: any) {
			this.data(null)
			return next ?? null
		}
	}
}
