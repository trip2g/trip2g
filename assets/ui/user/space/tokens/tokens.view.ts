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

		override token_rows() {
			if (this.data().size() === 0) {
				return [this.Empty()]
			}

			return this.data().map(key => this.Row(key))
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

		override row_expires_at(id: any) {
			return this.row(id).expiresAt
		}

		override row_revoked_at(id: any) {
			return this.row(id).revokedAt
		}
	}
}
