namespace $.$$ {
	export class $trip2g_user_space_tokens extends $.$trip2g_user_space_tokens {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_user_space_tokens_list()
			if (!res.viewer.user) {
				return $trip2g_graphql_make_map([])
			}
			return $trip2g_graphql_make_map(res.viewer.user.tokens)
		}

		@$mol_mem
		override spreads(): any {
			return {
				add: this.AddForm(),
				...this.data().mapKeys(key => this.Content(key)),
			}
		}

		// "add" is a screen, not a token: it is reachable by the New link and has
		// no place in the list of tokens the user holds.
		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter(id => id !== 'add')
		}

		row(id: any) {
			return this.data().get(id)
		}

		override row_id(id: any): string {
			return this.row(id).id
		}

		override row_name(id: any): string {
			return this.row(id).name
		}

		override row_prefix(id: any): string {
			return this.row(id).tokenPrefix
		}

		override row_created_at(id: any) {
			return this.row(id).createdAt
		}

		override row_last_used_at(id: any) {
			return this.row(id).lastUsedAt
		}

		override row_expires_at(id: any) {
			return this.row(id).expiresAt
		}

		// The revoke button reports back through this, and the list reloads:
		// a revoked token stays in the list, it just stops working.
		override row_revoked(id: any, next?: any) {
			this.data(null)
			return next ?? null
		}
	}
}
