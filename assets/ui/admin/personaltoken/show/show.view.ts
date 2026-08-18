namespace $.$$ {
	export class $trip2g_admin_personaltoken_show extends $.$trip2g_admin_personaltoken_show {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_admin_personaltoken_show_show({ id: this.token_id() })
			return res.admin.personalToken
		}

		moment(value: string | null | undefined): string {
			if (!value) {
				return '-'
			}

			return new $mol_time_moment(value).toString('YYYY-MM-DD hh:mm:ss')
		}

		override token_name(): string {
			return this.data()?.name ?? '-'
		}

		override token_owner(): string {
			return this.data()?.user?.email ?? '-'
		}

		override token_prefix(): string {
			return this.data()?.tokenPrefix ?? '-'
		}

		override token_scope(): string {
			return this.data()?.scope ?? '-'
		}

		override token_created_at(): string {
			return this.moment(this.data()?.createdAt)
		}

		override token_last_used_at(): string {
			return this.moment(this.data()?.lastUsedAt)
		}

		override token_expires_at(): string {
			return this.moment(this.data()?.expiresAt)
		}

		override token_revoked_at(): string {
			return this.moment(this.data()?.revokedAt)
		}

		// Revoking rewrites the row this page reads, so the page reloads itself
		// rather than showing a state the server no longer agrees with.
		override revoked(next?: any) {
			this.data(null)
			return next ?? null
		}
	}
}
